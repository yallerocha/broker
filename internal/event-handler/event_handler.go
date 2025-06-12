package eventhandler

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	"github.com/go-gota/gota/dataframe"
)

// Entrypoint to run the event handler.
// It receives a dataframe containing the data
func Handler(origin_data dataframe.DataFrame) {
	dynamicContext, err := utils.GetDynamicContext()

	utils.Log_fatal("Failed to get the client context", err)

	start_time := time.Now()
	df := origin_data.Arrange(dataframe.Sort("timestamp"))
	rows := df.Records()

	for i := range rows[0:df.Nrow()] {
		kind := df.Col("kind").Elem(i).String()
		time_stamp, err := df.Col("timestamp").Elem(i).Int()
		utils.Log_err("Failed to read the timestamp", err)

		sleep_time(time_stamp, start_time)

		if strings.ToLower(kind) == "deployment" {
			Deployment_action(dynamicContext, df, i)
		} else if strings.ToLower(kind) == "job" {
			Job_action(dynamicContext, df, i)
		} else if strings.ToLower(kind) == "node" {
			Node_action(dynamicContext, df, i)
		} else {
			utils.Log_err(fmt.Sprintf("Unknown kind '%s'", kind), fmt.Errorf("dataframe line %d", i+2))
		}

	}

}

// This function is responsible for synchronizing the broker execution time
// with the 'time_stamp' defined in the dataframe. If the execution time is less than
// the 'time_stamp', it will apply a sleep using the difference between
// the execution time and the 'time_stamp'.
// Receives the current 'time_stamp' in the dataframe and the 'start_time'.
func sleep_time(time_stamp int, start_time time.Time) {
	elapsed := time.Since(start_time)

	if int64(math.Ceil(elapsed.Seconds())) < int64(time_stamp) {
		time_wait := float64(time_stamp) - elapsed.Seconds()

		utils.Log_info(fmt.Sprintf("⏳ Waiting %d seconds", int64(time_wait)))
		time.Sleep(time.Duration(time_wait) * time.Second)
	}

}
