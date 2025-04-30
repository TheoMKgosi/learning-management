package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// getReviewHandler selects a due card for review in a given deck and renders an HTML review page.
// A card is considered due if its next_review is less than or equal to the current time.
func getReviewHandler(c *gin.Context) {
	deckID, err := strconv.Atoi(c.Param("deckID"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid deck ID")
		return
	}

	now := time.Now().Format(time.RFC3339)
	// Select one due card (if any); you might want to order by next_review ascending.
	row := db.QueryRow(
		"SELECT id, deck_id, front, back, interval, ease, last_review, next_review FROM cards WHERE deck_id = ? AND (next_review <= ? OR next_review IS NULL) ORDER BY next_review ASC LIMIT 1",
		deckID, now,
	)
	var card Card
	err = row.Scan(&card.ID, &card.DeckID, &card.Front, &card.Back, &card.Interval, &card.Ease, &card.LastReview, &card.NextReview)
	if err != nil {
		// If no card is due, render a page indicating so.
		c.HTML(http.StatusOK, "review.html", gin.H{"message": "No cards are due for review right now.", "card": nil})
		return
	}

	c.HTML(http.StatusOK, "review.html", gin.H{"card": card, "message": ""})
}

// postReviewHandler processes the review feedback submitted from the HTML form.  
// It expects two form fields: "card_id" and "rating" ("good" or "again").
// If "good", we update the card’s interval by multiplying by the ease factor.
// If "again", we reset the interval to 1 day.
func postReviewHandler(c *gin.Context) {
	cardIDStr := c.PostForm("card_id")
	rating := c.PostForm("rating")
	cardID, err := strconv.Atoi(cardIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid card id")
		return
	}

	// Get current card data.
	var card Card
	row := db.QueryRow("SELECT id, deck_id, front, back, interval, ease, last_review, next_review FROM cards WHERE id = ?", cardID)
	if err := row.Scan(&card.ID, &card.DeckID, &card.Front, &card.Back, &card.Interval, &card.Ease, &card.LastReview, &card.NextReview); err != nil {
		c.String(http.StatusInternalServerError, "Card not found")
		return
	}
	now := time.Now()
	var newInterval int
	// Basic spaced repetition logic:
	if rating == "good" {
		// For simplicity, multiply the current interval by the ease factor.
		newInterval = int(float64(card.Interval) * card.Ease)
		// Optionally you can adjust the ease factor further here.
	} else { // rating "again"
		newInterval = 1
	}
	// Update the card:
	// - Set last_review to now.
	// - Set next_review to now + newInterval days.
	nextReviewTime := now.Add(time.Duration(newInterval*24) * time.Hour).Format(time.RFC3339)
	_, err = db.Exec("UPDATE cards SET interval = ?, last_review = ?, next_review = ? WHERE id = ?",
		newInterval, now.Format(time.RFC3339), nextReviewTime, card.ID)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// Redirect back to review page.
	c.Redirect(http.StatusSeeOther, "/decks/"+strconv.Itoa(card.DeckID)+"/review")
}
