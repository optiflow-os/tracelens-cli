package goal

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/optiflow-os/tracelens-cli/pkg/utils"
)

type (
	// Goal 表示一个目标。
	Goal struct {
		CachedAt string `json:"cached_at"`
		Data     Data   `json:"data"`
	}

	// Range 表示目标的时间范围。
	Range struct {
		Date     string `json:"date"`
		End      string `json:"end"`
		Start    string `json:"start"`
		Text     string `json:"text"`
		Timezone string `json:"timezone"`
	}

	// ChartData 表示目标的图表数据。
	ChartData struct {
		ActualSeconds          float64 `json:"actual_seconds"`
		ActualSecondsText      string  `json:"actual_seconds_text"`
		GoalSeconds            int     `json:"goal_seconds"`
		GoalSecondsText        string  `json:"goal_seconds_text"`
		Range                  Range   `json:"range"`
		RangeStatus            string  `json:"range_status"`
		RangeStatusReason      string  `json:"range_status_reason"`
		RangeStatusReasonShort string  `json:"range_status_reason_short"`
	}

	// Owner 表示目标的所有者。
	Owner struct {
		DisplayName string  `json:"display_name"`
		Email       *string `json:"email"`
		FullName    string  `json:"full_name"`
		ID          string  `json:"id"`
		Photo       string  `json:"photo"`
		Username    string  `json:"username"`
	}

	// Subscriber 表示目标的订阅者。
	Subscriber struct {
		DisplayName    string  `json:"display_name"`
		Email          *string `json:"email"`
		EmailFrequency string  `json:"email_frequency"`
		FullName       string  `json:"full_name"`
		UserID         string  `json:"user_id"`
		Username       string  `json:"username"`
	}

	// Data 表示目标的数据。
	Data struct {
		AverageStatus           string       `json:"average_status"`
		ChartData               []ChartData  `json:"chart_data"`
		CreatedAt               string       `json:"created_at"`
		CumulativeStatus        string       `json:"cumulative_status"`
		CustomTitle             *string      `json:"custom_title"`
		Delta                   string       `json:"delta"`
		Editors                 []string     `json:"editors"`
		ID                      string       `json:"id"`
		IgnoreDays              []string     `json:"ignore_days"`
		IgnoreZeroDays          bool         `json:"ignore_zero_days"`
		ImproveByPercent        *float64     `json:"improve_by_percent"`
		IsCurrentUserOwner      bool         `json:"is_current_user_owner"`
		IsEnabled               bool         `json:"is_enabled"`
		IsInverse               bool         `json:"is_inverse"`
		IsSnoozed               bool         `json:"is_snoozed"`
		IsTweeting              bool         `json:"is_tweeting"`
		Languages               []string     `json:"languages"`
		ModifiedAt              *string      `json:"modified_at"`
		Owner                   Owner        `json:"owner"`
		Projects                []string     `json:"projects"`
		RangeText               string       `json:"range_text"`
		Seconds                 int          `json:"seconds"`
		SharedWith              []string     `json:"shared_with"`
		SnoozeUntil             *string      `json:"snooze_until"`
		Status                  string       `json:"status"`
		StatusPercentCalculated int          `json:"status_percent_calculated"`
		Subscribers             []Subscriber `json:"subscribers"`
		Title                   string       `json:"title"`
		Type                    string       `json:"type"`
	}
)

// RenderToday 从当前日期的目标生成文本表示。
// 如果out设置为output.RawJSONOutput或output.JSONOutput，目标将被序列化为JSON。
// 预期当前日期恰好有一个摘要。否则将返回错误。
func RenderToday(goal *Goal, out utils.Output) (string, error) {
	if goal == nil {
		return "", errors.New("no goal found for the current day")
	}

	if len(goal.Data.ChartData) == 0 {
		return "", errors.New("no chart data found for the current day")
	}

	if out == utils.RawJSONOutput {
		data, err := json.Marshal(goal)
		if err != nil {
			return "", fmt.Errorf("failed to marshal json goal: %s", err)
		}

		return string(data), nil
	}

	return goal.Data.ChartData[len(goal.Data.ChartData)-1].ActualSecondsText, nil
}
