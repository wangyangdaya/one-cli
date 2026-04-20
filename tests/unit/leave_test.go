package unit_test

import (
	"testing"

	"one-cli/examples/one-leave/leave"
)

func TestMaskQuotaForDisplayRemovesQuotaForTimeOffInLieu(t *testing.T) {
	items := []leave.LeaveInfo{
		{
			VacationTypeName:   "带薪年假",
			VacationType:       "annual_leave",
			ReadableQuotaValue: "5天",
		},
		{
			VacationTypeName:   "调休",
			VacationType:       "day_off",
			ReadableQuotaValue: "8小时",
		},
	}

	got := leave.MaskQuotaForDisplay(items)

	if got[0].ReadableQuotaValue != "5天" {
		t.Fatalf("expected non-调休 quota to be preserved, got %q", got[0].ReadableQuotaValue)
	}
	if got[1].ReadableQuotaValue != "" {
		t.Fatalf("expected 调休 quota to be masked, got %q", got[1].ReadableQuotaValue)
	}
	if items[1].ReadableQuotaValue != "8小时" {
		t.Fatalf("expected input slice to remain unmodified, got %q", items[1].ReadableQuotaValue)
	}
}

func TestFilterByLeaveTypeMatchesCodeOrName(t *testing.T) {
	items := []leave.LeaveInfo{
		{VacationTypeName: "带薪年假", VacationType: "annual_leave"},
		{VacationTypeName: "调休", VacationType: "day_off"},
	}

	byCode := leave.FilterByLeaveType(items, "annual_leave")
	if len(byCode) != 1 || byCode[0].VacationType != "annual_leave" {
		t.Fatalf("expected code filter to return annual_leave, got %#v", byCode)
	}

	byName := leave.FilterByLeaveType(items, "调休")
	if len(byName) != 1 || byName[0].VacationTypeName != "调休" {
		t.Fatalf("expected name filter to return 调休, got %#v", byName)
	}
}
