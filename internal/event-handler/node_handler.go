package eventhandler

import (
	"fmt"

	"github.com/cloud-ai-ufcg/broker/pkg/utils"
	"github.com/go-gota/gota/dataframe"
	"k8s.io/client-go/dynamic"
)

// This function identifies the specific action related to a given node and
// performs the action.
// It receives a dynamic context to perform the requests, a DataFrame containing the
// data and the 'idx' representing the node index in the DataFrame.
func Node_action(dynamicContext *dynamic.DynamicClient, df dataframe.DataFrame, idx int) {
	utils.Log_err("Node action not implemented!", fmt.Errorf(""))
}
