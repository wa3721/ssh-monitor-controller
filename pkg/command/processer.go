package command

import (
	"k8s.io/apimachinery/pkg/util/json"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"ssh-monitor-controller/api/v1alpha1"
)

const (
	errorLogPath = "/var/log/ssh-monitor-controller-error.log"
)

func processCommand(cmd *string, ssE *v1alpha1.SshCommandEntity) error {
	log := ctrl.Log.WithName("processCommand")
	err := json.Unmarshal([]byte(*cmd), ssE)
	if err != nil {
		log.Error(err, "unmarshal command error", "cmd = ", *cmd, "this command will be skipped and write into error log...")
		err = os.WriteFile(errorLogPath, []byte("unmarshal failed: "+*cmd+"\n"), os.ModePerm)
		if err != nil {
			log.Error(err, "write error command to error log... ", "cmd = ", *cmd)
		}
		return err
	}
	return nil
}
