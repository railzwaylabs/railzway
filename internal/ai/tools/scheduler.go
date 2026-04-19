package tools

import (
	"fmt"
	"time"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/railzwaylabs/railzway/internal/ai/scheduler"
	"github.com/railzwaylabs/railzway/internal/orgcontext"
)

type scheduleJobInput struct {
	TaskType     string      `json:"task_type" desc:"Type of task (e.g. reminder, export_report)"`
	ScheduleCron string      `json:"schedule_cron,omitempty" desc:"Optional cron schedule for recurring tasks"`
	RunInMinutes int         `json:"run_in_minutes,omitempty" desc:"Optional delay in minutes for one-off tasks"`
	Payload      map[string]any `json:"payload" desc:"JSON parameters for the task"`
}

func defineSchedulerTools(g *genkit.Genkit, s *scheduler.Service) []ai.ToolRef {
	if s == nil {
		return nil
	}

	scheduleJobTool := genkit.DefineTool(
		g,
		"ai_assistant_schedule_job",
		"Schedules a background task or reminder.",
		func(ctx *ai.ToolContext, input scheduleJobInput) (string, error) {
			orgID, ok := orgcontext.OrgIDFromContext(ctx.Context)
			if !ok {
				return "", fmt.Errorf("missing_organization_context")
			}

			inputData := scheduler.CreateJobInput{
				OrgID:        orgID,
				TaskType:     input.TaskType,
				Payload:      input.Payload,
				ScheduleCron: input.ScheduleCron,
			}

			if input.RunInMinutes > 0 {
				runAt := time.Now().UTC().Add(time.Duration(input.RunInMinutes) * time.Minute)
				inputData.RunAt = &runAt
			}

			job, err := s.CreateJob(ctx.Context, inputData)
			if err != nil {
				return "", err
			}

			nextRunStr := job.NextRunAt.Format("2006-01-02 15:04:05 MST")
			return fmt.Sprintf("Successfully scheduled job %s (%s). Next run at %s.", job.ID, job.TaskType, nextRunStr), nil
		},
	)

	return []ai.ToolRef{scheduleJobTool}
}
