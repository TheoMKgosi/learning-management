package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// getDecksHandler returns all decks as JSON.
func getDecksHandler(c *gin.Context) {
	rows, err := db.Query("SELECT id, name FROM decks")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var decks []Deck
	for rows.Next() {
		var d Deck
		if err := rows.Scan(&d.ID, &d.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		decks = append(decks, d)
	}
	c.JSON(http.StatusOK, decks)
}

// createDeckHandler creates a new deck.
func createDeckHandler(c *gin.Context) {
	var d Deck
	if err := c.ShouldBind(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	if d.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Deck name is required"})
		return
	}

	result, err := db.Exec("INSERT INTO decks (name) VALUES (?)", d.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	insertID, _ := result.LastInsertId()
	d.ID = int(insertID)
	c.JSON(http.StatusOK, d)
}

func editDeckHandler(c *gin.Context)  {
  deckID, err := strconv.Atoi(c.Param("deckID"))  
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deck ID"})
    return
  }

  var deck Deck
  if err := c.ShouldBind(&deck); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
    return
  }
  if deck.Name == "" {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
    return
  }

  _, err = db.Exec("UPDATE decks SET name = ? WHERE id = ?", deck.Name , deckID)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Deck update unsuccessful"})
    return
  }
  c.JSON(http.StatusOK, gin.H{"message": "Deck updated successfully"})
}

func deleteDeckHandler(c *gin.Context)  {
  deckID, err := strconv.Atoi(c.Param("deckID"))  
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deck ID"})
    return
  }

  _, err = db.Exec("DELETE FROM decks WHERE id = ?", deckID)
  if err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Deck delete unsuccessful"})
    return
  }
  c.JSON(http.StatusOK, gin.H{"meesage": "Deck deleted successfully"})

  
}
