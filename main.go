package main

import (
  "database/sql"
  "fmt"
  "log"
  "net/http"
  "strconv"
  "time"

  "github.com/gin-gonic/gin"
  _ "github.com/mattn/go-sqlite3"
)

// Hardcoded login credentials for demonstration.
const (
  hardcodedUser     = "admin"
  hardcodedPassword = "secret"
  // Name of the cookie used for our simple session.
  sessionCookieName = "username"
)

// A Deck groups flashcards.
type Deck struct {
  ID   int
  Name string
}

// A Card is a flashcard. We add two extra fields for spaced repetition:
// Interval: current review interval (in days)
// NextReview: the next time at which the card is due.
type Card struct {
  ID         int
  Question   string
  Answer     string
  DeckID     int
  DeckName   string // populated via join for display purposes
  Interval   int    // in minutes (0 means not yet set)
  NextReview time.Time
}

var db *sql.DB

// initDB creates the necessary tables if they don’t exist.
func initDB() {
  var err error
  db, err = sql.Open("sqlite3", "./anki.db")
  if err != nil {
    log.Fatal("Error opening database: ", err)
  }

  // Create decks table.
  deckStmt := `
  CREATE TABLE IF NOT EXISTS decks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL
  );`
  _, err = db.Exec(deckStmt)
  if err != nil {
    log.Fatal("Error creating decks table: ", err)
  }

  // Create cards table.
  // interval: number of days for next repetition (0 = new card).
  // next_review: when the card is due for review.
  cardStmt := `
  CREATE TABLE IF NOT EXISTS cards (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  question TEXT NOT NULL,
  answer TEXT NOT NULL,
  deck_id INTEGER NOT NULL,
  interval INTEGER DEFAULT 0,
  next_review DATETIME DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY(deck_id) REFERENCES decks(id)
  );`
  _, err = db.Exec(cardStmt)
  if err != nil {
    log.Fatal("Error creating cards table: ", err)
  }
}

// requireLogin is a simple middleware that checks for our cookie.
func requireLogin(c *gin.Context) {
  user, err := c.Cookie(sessionCookieName)
  if err != nil || user != hardcodedUser {
    c.Redirect(http.StatusFound, "/login")
    c.Abort()
    return
  }
  c.Next()
}

func main() {
  initDB()

  router := gin.Default()
  router.Static("/static", "./static")
  router.LoadHTMLGlob("templates/*")

  // Public routes.
  router.GET("/login", func(c *gin.Context) {
    c.HTML(http.StatusOK, "login.html", nil)
  })
  router.POST("/login", func(c *gin.Context) {
    username := c.PostForm("username")
    password := c.PostForm("password")
    if username == hardcodedUser && password == hardcodedPassword {
      // set cookie for simple session management (expires in 1 hour)
      c.SetCookie(sessionCookieName, username, 3600, "/", "", false, true)
      c.Redirect(http.StatusFound, "/decks")
      return
    }
    c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Invalid credentials"})
  })
  router.GET("/logout", func(c *gin.Context) {
    c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
    c.Redirect(http.StatusFound, "/login")
  })

  // Group routes that require login.
  protected := router.Group("/")
  protected.Use(requireLogin)
  {
    // List all decks.
    protected.GET("/decks", func(c *gin.Context) {
      rows, err := db.Query("SELECT id, name FROM decks ORDER BY id DESC")
      if err != nil {
        c.String(http.StatusInternalServerError, err.Error())
        return
      }
      defer rows.Close()

      var decks []Deck
      for rows.Next() {
        var d Deck
        if err := rows.Scan(&d.ID, &d.Name); err != nil {
          c.String(http.StatusInternalServerError, err.Error())
          return
        }
        decks = append(decks, d)
      }
      c.HTML(http.StatusOK, "deck_list.html", gin.H{"decks": decks})
    })

    // Form to create a new deck.
    protected.GET("/decks/new", func(c *gin.Context) {
      c.HTML(http.StatusOK, "new_deck.html", nil)
    })
    protected.POST("/decks", func(c *gin.Context) {
      name := c.PostForm("name")
      _, err := db.Exec("INSERT INTO decks (name) VALUES (?)", name)
      if err != nil {
        c.String(http.StatusInternalServerError, err.Error())
        return
      }
      c.Redirect(http.StatusFound, "/decks")
    })

    // View a single deck (list its cards) and option to add a card.
    protected.GET("/decks/:deckID", func(c *gin.Context) {
      deckID, err := strconv.Atoi(c.Param("deckID"))
      if err != nil {
        c.String(http.StatusBadRequest, "Invalid deck id")
        return
      }
      // Get deck details.
      var deck Deck
      err = db.QueryRow("SELECT id, name FROM decks WHERE id = ?", deckID).Scan(&deck.ID, &deck.Name)
      if err != nil {
        c.String(http.StatusInternalServerError, err.Error())
        return
      }
      // Get cards in deck.
      rows, err := db.Query("SELECT id, question, answer, interval, next_review FROM cards WHERE deck_id = ? ORDER BY id DESC", deckID)
      if err != nil {
        c.String(http.StatusInternalServerError, err.Error())
        return
      }
      defer rows.Close()
      var cards []Card
      for rows.Next() {
        var card Card
        var nextReview string
        if err := rows.Scan(&card.ID, &card.Question, &card.Answer, &card.Interval, &nextReview); err != nil {
          c.String(http.StatusInternalServerError, err.Error())
          return
        }
        card.DeckID = deckID
        // Parse next_review as time.Time.
        card.NextReview, _ = time.Parse("2006-01-02 15:04:05", nextReview)
        cards = append(cards, card)
      }
      c.HTML(http.StatusOK, "deck.html", gin.H{
        "deck":  deck,
        "cards": cards,
      })
    })

    // Form to add a new card for a given deck.
    protected.GET("/decks/:deckID/cards/new", func(c *gin.Context) {
      deckID, err := strconv.Atoi(c.Param("deckID"))
      if err != nil {
        c.String(http.StatusBadRequest, "Invalid deck id")
        return
      }
      c.HTML(http.StatusOK, "new_card.html", gin.H{"deckID": deckID})
    })

    protected.POST("/decks/:deckID/cards", func(c *gin.Context) {
      deckID, err := strconv.Atoi(c.Param("deckID"))
      if err != nil {
        c.String(http.StatusBadRequest, "Invalid deck id")
        return
      }
      question := c.PostForm("question")
      answer := c.PostForm("answer")
      now := time.Now().Format("2006-01-02 15:04:05")
      // interval is 0 for new cards
      _, err = db.Exec("INSERT INTO cards (question, answer, deck_id, interval, next_review) VALUES (?, ?, ?, 0, ?)",
        question, answer, deckID, now)
      if err != nil {
        c.String(http.StatusInternalServerError, err.Error())
        return
      }
      c.Redirect(http.StatusFound, fmt.Sprintf("/decks/%d", deckID))
    })

    // Review cards for a given deck.
    // Only cards that are due (i.e. next_review <= now) will be shown.
    protected.GET("/decks/:deckID/review", func(c *gin.Context) {
      deckID, err := strconv.Atoi(c.Param("deckID"))
      if err != nil {
        c.String(http.StatusBadRequest, "Invalid deck id")
        return
      }
      now := time.Now().Format("2006-01-02 15:04:05")
      rows, err := db.Query("SELECT id, question, answer, interval, next_review FROM cards WHERE deck_id = ? AND next_review <= ? ORDER BY next_review ASC", deckID, now)
      if err != nil {
        c.String(http.StatusInternalServerError, err.Error())
        return
      }
      defer rows.Close()
      var cards []Card
      for rows.Next() {
        var card Card
        var nextReview string
        if err := rows.Scan(&card.ID, &card.Question, &card.Answer, &card.Interval, &nextReview); err != nil {
          c.String(http.StatusInternalServerError, err.Error())
          return
        }
        card.DeckID = deckID
        card.NextReview, _ = time.Parse("2006-01-02 15:04:05", nextReview)
        cards = append(cards, card)
      }
      c.HTML(http.StatusOK, "review.html", gin.H{
        "deckID": deckID,
        "cards":  cards,
      })
    })

    // Review a card using an intense spaced repetition algorithm.
    // Our time units are in hours.
    // For an "easy" rating, if the card is new, set interval = 1 hour; else multiply by 1.5.
    // For a "hard" rating, if the card is new, set interval = 0.5 hour; else multiply by 1.2.
    protected.POST("/cards/:cardID/review", func(c *gin.Context) {
      cardID, err := strconv.Atoi(c.Param("cardID"))
      if err != nil {
        c.String(http.StatusBadRequest, "Invalid card id")
        return
      }
      rating := c.PostForm("rating")
      // Retrieve current interval (in hours) and deck id for redirection.
      var interval float64
      var deckID int
      err = db.QueryRow("SELECT interval, deck_id FROM cards WHERE id = ?", cardID).Scan(&interval, &deckID)
      if err != nil {
        c.String(http.StatusInternalServerError, err.Error())
        return
      }

      var newInterval float64
      if rating == "easy" {
        if interval == 0 {
          newInterval = 1.0 // 1 hour for new cards rated easy
        } else {
          newInterval = interval * 1.5
        }
      } else { // "hard" rating
        if interval == 0 {
          newInterval = 0.5 // 30 minutes for new cards rated hard
        } else {
          newInterval = interval * 1.2
        }
      }

      // Compute the next review time, adding newInterval hours.
      nextReview := time.Now().Add(time.Duration(newInterval * float64(time.Hour)))
      // Update the card record: store the new interval as a float (but our database field is an integer)
      // To support fractional hours in the database, you might store the interval value as a REAL.
      //
      // For this demo, we convert hours to an integer number of minutes.
      // For example, 1.5 hours becomes 90 minutes.
      newIntervalMins := int(newInterval * 60)
      _, err = db.Exec("UPDATE cards SET interval = ?, next_review = ? WHERE id = ?", newIntervalMins, nextReview.Format("2006-01-02 15:04:05"), cardID)
      if err != nil {
        c.String(http.StatusInternalServerError, err.Error())
        return
      }
      c.Redirect(http.StatusFound, fmt.Sprintf("/decks/%d/review", deckID))
    })
  }

  router.Run(":8080")
}
