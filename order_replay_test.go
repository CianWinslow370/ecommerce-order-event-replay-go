package main

import "testing"

func TestReplayRequiresTheRightTarget(t *testing.T) {
	cases := []struct {
		name string
		in   ReplayRequest
		want string
	}{
		{"platform order", ReplayRequest{Source: "platform", OrderID: "ord-17", WebhookID: "wh-2"}, "delivery_history_read"},
		{"own dead letter", ReplayRequest{Source: "dead_letter", DLQ: "checkout-dlq"}, "dead_letter_redriven"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := replayAction(tc.in.Source)
			if err != nil || got != tc.want {
				t.Fatalf("replayAction(%q) = %q, %v; want %q", tc.in.Source, got, err, tc.want)
			}
		})
	}
}
