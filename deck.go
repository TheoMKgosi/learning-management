package main

import (
	"net/http"

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
