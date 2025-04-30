package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// getCardsHandler returns cards for a specific deck.
func getCardsHandler(c *gin.Context) {
	deckID, err := strconv.Atoi(c.Param("deckID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deck ID"})
		return
	}

	rows, err := db.Query("SELECT id, deck_id, front, back, interval, ease, last_review, next_review FROM cards WHERE deck_id = ?", deckID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var cards []Card
	for rows.Next() {
		var card Card
		if err := rows.Scan(&card.ID, &card.DeckID, &card.Front, &card.Back, &card.Interval, &card.Ease, &card.LastReview.String, &card.NextReview.String); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		cards = append(cards, card)
	}
	c.JSON(http.StatusOK, cards)
}

// createCardHandler creates a new card in a deck, initializing spaced repetition fields.
func createCardHandler(c *gin.Context) {
	deckID, err := strconv.Atoi(c.Param("deckID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deck ID"})
		return
	}

	var card Card
	if err := c.ShouldBind(&card); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	if card.Front == "" || card.Back == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Both front and back of the card are required"})
		return
	}

	// For new cards: default interval=1 day, ease=2.5 and schedule next_review as now.
	now := time.Now().Format(time.RFC3339)
	result, err := db.Exec(
		"INSERT INTO cards (deck_id, front, back, interval, ease, next_review) VALUES (?, ?, ?, 1, 2.5, ?)",
		deckID, card.Front, card.Back, now,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	insertID, _ := result.LastInsertId()
	card.ID = int(insertID)
	card.DeckID = deckID
	card.Interval = 1
	card.Ease = 2.5
	card.NextReview.String = now
	c.JSON(http.StatusOK, card)
}
