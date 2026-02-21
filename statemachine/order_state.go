package statemachine

import (
	"errors"
	"food-delivery-api/models"
)

// Transition defines a valid state change and who can perform it
type Transition struct {
	From    models.OrderStatus
	To      models.OrderStatus
	Actor   string // "restaurant", "driver", "customer", "system"
}

// validTransitions is the authoritative state machine definition
var validTransitions = []Transition{
	// Restaurant confirms the order
	{From: models.StatusPlaced, To: models.StatusConfirmed, Actor: "restaurant"},
	// Restaurant or Customer can cancel a PLACED order
	{From: models.StatusPlaced, To: models.StatusCancelled, Actor: "restaurant"},
	{From: models.StatusPlaced, To: models.StatusCancelled, Actor: "customer"},
	// Restaurant or Customer can cancel a CONFIRMED order
	{From: models.StatusConfirmed, To: models.StatusPreparing, Actor: "restaurant"},
	{From: models.StatusConfirmed, To: models.StatusCancelled, Actor: "restaurant"},
	{From: models.StatusConfirmed, To: models.StatusCancelled, Actor: "customer"},
	// Restaurant marks order ready for pickup
	{From: models.StatusPreparing, To: models.StatusReadyForPickup, Actor: "restaurant"},
	// Driver picks up the order
	{From: models.StatusReadyForPickup, To: models.StatusPickedUp, Actor: "driver"},
	// Driver delivers the order
	{From: models.StatusPickedUp, To: models.StatusDelivered, Actor: "driver"},
}

// transitionKey is used to look up valid transitions quickly
type transitionKey struct {
	From  models.OrderStatus
	To    models.OrderStatus
	Actor string
}

// Build a lookup map for O(1) validation
var transitionMap = func() map[transitionKey]bool {
	m := make(map[transitionKey]bool)
	for _, t := range validTransitions {
		m[transitionKey{t.From, t.To, t.Actor}] = true
	}
	return m
}()

// ValidTransitionsFrom returns all valid next states from a given state
func ValidTransitionsFrom(status models.OrderStatus) []models.OrderStatus {
	var nexts []models.OrderStatus
	seen := map[models.OrderStatus]bool{}
	for _, t := range validTransitions {
		if t.From == status && !seen[t.To] {
			nexts = append(nexts, t.To)
			seen[t.To] = true
		}
	}
	return nexts
}

// CanTransition checks if a given actor can move from one state to another
func CanTransition(from, to models.OrderStatus, actor string) error {
	key := transitionKey{From: from, To: to, Actor: actor}
	if transitionMap[key] {
		return nil
	}
	return errors.New(
		"invalid transition: " + string(from) + " → " + string(to) +
			" is not allowed for actor '" + actor + "'. " +
			"Valid transitions from " + string(from) + " are: " + describeValidFrom(from),
	)
}

func describeValidFrom(status models.OrderStatus) string {
	nexts := ValidTransitionsFrom(status)
	if len(nexts) == 0 {
		return "none (terminal state)"
	}
	result := ""
	for i, s := range nexts {
		if i > 0 {
			result += ", "
		}
		result += string(s)
	}
	return result
}

// GetAllTransitions returns the full state machine for documentation
func GetAllTransitions() []Transition {
	return validTransitions
}
