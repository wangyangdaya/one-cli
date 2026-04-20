package unit_test

import (
	"testing"

	"one-cli/examples/one-leave/leave"
)

func TestValidateRequestRequiresAppendixForSickLeave(t *testing.T) {
	err := leave.ValidateRequest(leave.RequestRequest{
		JobNo:         "415327",
		StartTime:     "2026-03-25 09:00",
		EndTime:       "2026-03-25 18:00",
		VacationType:  "sick_leave",
		LeaveTimeType: 0,
		Reason:        "病假",
	})
	if err == nil {
		t.Fatal("expected appendix requirement validation error")
	}
}
