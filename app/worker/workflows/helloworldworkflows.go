package workflows

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

/**
 * This is the hello world workflow sample.
 */

// ApplicationName is the task list for this sample
const TaskListName = "helloWorldGroup"
const SignalName = "helloWorldSignal"

// This is registration process where you register all your workflows
// and activity function handlers.
func init() {
	workflow.Register(Workflow)
	activity.Register(helloworldActivity)
	activity.Register(maxAgeActivity)
}

var activityOptions = workflow.ActivityOptions{
	ScheduleToStartTimeout: time.Minute,
	StartToCloseTimeout:    time.Minute,
	HeartbeatTimeout:       time.Second * 20,
}

func helloworldActivity(ctx context.Context, name string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("helloworld activity started")
	return "Hello " + name + "! How old are you!", nil
}

func maxAgeActivity(ctx context.Context) (int, error) {
	return rand.Intn(100), nil
}

func Workflow(ctx workflow.Context, name string) (string, error) {
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	logger := workflow.GetLogger(ctx)
	logger.Info("helloworld workflow started")
	var activityResult string
	err := workflow.ExecuteActivity(ctx, helloworldActivity, name).Get(ctx, &activityResult)
	if err != nil {
		logger.Error("Activity failed.", zap.Error(err))
		return "", err
	}

	// After saying hello, the workflow will wait for you to inform it of your age!
	selector := workflow.NewSelector(ctx)
	var ageResult int
	var maxAge = 150

	for {
		err := workflow.ExecuteActivity(ctx, maxAgeActivity).Get(ctx, &maxAge)
		if err != nil {
			logger.Error("Activity failed.", zap.Error(err))
			return "", err
		}

		selector.AddReceive(workflow.GetSignalChannel(ctx, SignalName), func(c workflow.Channel, more bool) {
			c.Receive(ctx, &ageResult)
			workflow.GetLogger(ctx).Info("Received age results from signal!", zap.String("signal", SignalName), zap.Int("value", ageResult))
		})
		workflow.GetLogger(ctx).Info("Waiting for signal on channel.. " + SignalName)
		// Wait for signal
		selector.Select(ctx)

		// We can check the age and return an appropriate response
		if ageResult > 0 && ageResult < maxAge {
			logger.Info("Workflow completed.", zap.String("NameResult", activityResult), zap.Int("AgeResult", ageResult))

			return fmt.Sprintf("Hello "+name+"! You are %v years old!", ageResult), nil
		} else {
			return "You can't be that old!", nil
		}
	}
}
