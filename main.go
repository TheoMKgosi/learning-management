package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

// Hard-coded credentials for basic authentication.
const (
  authUsername = "admin"
  authPassword = "secret"
)

var db *sql.DB

func main() {
  var err error
  // Open (or create) the SQLite database.
  db, err = sql.Open("sqlite3", "./learning-management.db")
  if err != nil {
    panic("Error opening database: " + err.Error())
  }
  defer db.Close()

  // Initialize the database schema.
  if err := initDB(); err != nil {
    panic("Error initializing database: " + err.Error())
  }

  // Create a Gin router and configure HTML template loading.
  router := gin.Default()
  router.LoadHTMLGlob("templates/*")

  // Global middleware to require basic authentication.
  router.Use(basicAuthMiddleware())

  // HTML endpoints.
  router.GET("/", indexHandler)
  router.GET("/decks/:deckID", deckDetailHandler)
  router.GET("/cards/:cardID", cardDetailHandler)
  router.GET("/decks/:deckID/edit", deckEditDetailHandler)

  // JSON endpoints for managing decks and cards.
  router.GET("/decks", getDecksHandler)
  router.POST("/decks", createDeckHandler)
  router.GET("/decks/:deckID/cards", getCardsHandler)
  router.POST("/decks/:deckID/cards", createCardHandler)
  router.POST("/cards/:cardID", editCardHandler)
  router.POST("/decks/:deckID", editDeckHandler)
  router.DELETE("/decks/:deckID", deleteDeckHandler)
  router.DELETE("/cards/:cardID", deleteCardHandler)

  // Spaced repetition review endpoints:
  // GET: render a review page that picks one due card.
  // POST: process the review result.
  router.GET("/decks/:deckID/review", getReviewHandler)
  router.POST("/decks/:deckID/review", postReviewHandler)

  // Start the server.
  router.Run(":8080")
}

// basicAuthMiddleware enforces HTTP basic auth for all requests.
func basicAuthMiddleware() gin.HandlerFunc {
  return func(c *gin.Context) {
    user, pass, hasAuth := c.Request.BasicAuth()
    if !hasAuth || user != authUsername || pass != authPassword {
      c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
      c.AbortWithStatus(http.StatusUnauthorized)
      return
    }
    c.Next()
  }
}

// initDB creates the required tables (decks and cards).

/////////////////////////
// HTML Handlers
/////////////////////////

// indexHandler renders a page listing all decks.
func indexHandler(c *gin.Context) {
  rows, err := db.Query("SELECT id, name FROM decks")
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
  c.HTML(http.StatusOK, "index.html", gin.H{"decks": decks})
}

// deckDetailHandler renders a page for a specific deck where cards are listed.
func deckDetailHandler(c *gin.Context) {
  id, err := strconv.Atoi(c.Param("deckID"))
  if err != nil {
    c.String(http.StatusBadRequest, "Invalid deck ID")
    return
  }

  // Get deck information.
  var deck Deck
  err = db.QueryRow("SELECT id, name FROM decks WHERE id = ?", id).Scan(&deck.ID, &deck.Name)
  if err != nil {
    c.String(http.StatusNotFound, "Deck not found")
    return
  }

  // Get cards for this deck.
  rows, err := db.Query("SELECT id, deck_id, front, back, interval, ease, last_review, next_review FROM cards WHERE deck_id = ?", id)
  if err != nil {
    c.String(http.StatusInternalServerError, err.Error())
    return
  }
  defer rows.Close()

  var cards []Card
  for rows.Next() {
    var card Card
    if err := rows.Scan(&card.ID, &card.DeckID, &card.Front, &card.Back, &card.Interval, &card.Ease, &card.LastReview, &card.NextReview); err != nil {
      c.String(http.StatusInternalServerError, err.Error())
      return
    }
    cards = append(cards, card)
  }
  c.HTML(http.StatusOK, "deck.html", gin.H{
    "deck":  deck,
    "cards": cards,
  })
}

func deckEditDetailHandler(c *gin.Context)  {
  deckID, err := strconv.Atoi(c.Param("deckID"))
  if err != nil {
    log.Println(err)
    c.String(http.StatusBadRequest, "Invalid deck ID")
    return
  }

  var deck Deck
  err = db.QueryRow("SELECT id, name FROM decks WHERE id = ?", deckID).Scan(&deck.ID, &deck.Name)
  if err != nil {
    c.String(http.StatusBadRequest, "Something went wrong")
  }

  c.HTML(http.StatusOK, "deckEdit.html", gin.H{
    "deck": deck,
  })
}

func cardDetailHandler(c *gin.Context)  {
  id, err := strconv.Atoi(c.Param("cardID"))
  if err != nil {
    c.String(http.StatusBadRequest, "Invalid card ID")
    return
  }
  var card Card
  err = db.QueryRow("SELECT id, deck_id, front, back FROM cards WHERE id = ?", id).Scan(&card.ID, &card.DeckID, &card.Front, &card.Back)
  log.Println(card)
  if err != nil {
    c.String(http.StatusNotFound, "Card not found")
    return
  }

  c.HTML(http.StatusOK, "card.html", gin.H{
    "card": card,
  })
}

func initDB() error {
  // Create decks table.
  deckSchema := `
  CREATE TABLE IF NOT EXISTS decks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL
  );
  `
  _, err := db.Exec(deckSchema)
  if err != nil {
    return err
  }

  // Create cards table with fields for spaced repetition.
  cardSchema := `
  CREATE TABLE IF NOT EXISTS cards (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  deck_id INTEGER NOT NULL,
  front TEXT NOT NULL,
  back TEXT NOT NULL,
  interval INTEGER NOT NULL DEFAULT 1,
  ease REAL NOT NULL DEFAULT 2.5,
  last_review TEXT,
  next_review TEXT,
  FOREIGN KEY(deck_id) REFERENCES decks(id)
  );
  `
  _, err = db.Exec(cardSchema)
  return err
}
