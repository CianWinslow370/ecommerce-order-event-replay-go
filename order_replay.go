package main

import "fmt"

type ReplayRequest struct {
	OrderID   string `json:"order_id"`
	Source    string `json:"source"`
	WebhookID string `json:"webhook_id"`
	DLQ       string `json:"dlq"`
}

type ReplayResult struct {
	Action string `json:"action"`
	Target string `json:"target"`
}

func replayAction(source string) (string, error) {
	switch source {
	case "platform":
		return "delivery_history_read", nil
	case "dead_letter":
		return "dead_letter_redriven", nil
	default:
		return "", fmt.Errorf("source must be platform or dead_letter")
	}
}

func replay(client *InfraiClient, request ReplayRequest) (ReplayResult, error) {
	action, err := replayAction(request.Source)
	if err != nil {
		return ReplayResult{}, err
	}
	switch request.Source {
	case "platform":
		if request.WebhookID == "" || request.OrderID == "" {
			return ReplayResult{}, fmt.Errorf("platform replay needs order_id and webhook_id")
		}
		if _, err := client.DeliveryHistory(request.WebhookID, request.OrderID); err != nil {
			return ReplayResult{}, err
		}
		return ReplayResult{Action: action, Target: request.OrderID}, nil
	case "dead_letter":
		if request.DLQ == "" {
			return ReplayResult{}, fmt.Errorf("dead_letter replay needs dlq")
		}
		if _, err := client.Redrive(request.DLQ); err != nil {
			return ReplayResult{}, err
		}
		return ReplayResult{Action: action, Target: request.DLQ}, nil
	default:
		return ReplayResult{}, err
	}
}
