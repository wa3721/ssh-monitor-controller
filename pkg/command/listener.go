package command

import (
	"context"
	"ssh-monitor-controller/api/v1alpha1"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	commandListenerLog = ctrl.Log.WithName("CommandListener")
)

type Listener struct {
	client        client.Client
	scheme        *runtime.Scheme
	log           logr.Logger
	EventRecorder record.EventRecorder
}

func (l *Listener) ListenAndCreateOrUpdateCr() {
	for {
		select {
		case cmd := <-CmdChannel:
			var cEntity v1alpha1.SshCommandEntity
			err := json.Unmarshal([]byte(*cmd), &cEntity)
			if err != nil {
				commandListenerLog.Error(err, "Failed to unmarshal command")
			}
			// create cr
			//收到新的信号之后
			//先找到创建时间为当天，并且满足label上同时存在 node 和scrip同时相同的cr，如果没有创建一个新的，如果存在，更新存在的cr

		}
	}
}

func (l *Listener) findLabeledCrsToday(ctx context.Context, entity *v1alpha1.SshCommandEntity) (*v1alpha1.SshCommandAudit, error) {
	var allCrs v1alpha1.SshCommandAuditList
	//列出所有打过标签的cr
	err := l.client.List(ctx, &allCrs, client.MatchingLabels{nodeLabel: entity.Node, srcLabel: entity.SrcIp})
	if err != nil {
		commandListenerLog.Error(err, "Failed to list all labeled crs")
		return nil, err
	}
	//判断是否存在定义的annotation

	//判断是否是今天
	for _, cr := range allCrs.Items {
		if cr.CreationTimestamp.Time.Format(time.DateOnly) != time.Now().Format(time.DateOnly) {
			//不是今天创建的

		}
	}

}

func newSshCommandCr(entity *v1alpha1.SshCommandEntity) *v1alpha1.SshCommandAudit {
	return &v1alpha1.SshCommandAudit{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SshCommandAudit",
			APIVersion: v1alpha1.GroupVersion.String(),
		},
		//ObjectMeta
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "sshcommandaudit-",
			Annotations:  generateAnnotations(entity),
			Labels:       generateCrLabels(entity),
		},
		Spec: v1alpha1.SshCommandAuditSpec{
			SshCommandEntity: []*v1alpha1.SshCommandEntity{},
		},
	}
}

// updateSshCommandCrSpec update cr spec to record command

func (l *Listener) updateSshCommandCrSpec(ctx context.Context, cr *v1alpha1.SshCommandAudit, entity *v1alpha1.SshCommandEntity) error {
	commandListenerLog.Info("Updating SshCommand CR Spec...")
	l.client.Update(ctx, cr)

}

const (
	nodeLabel                   = "ssh.monitor.io/NodeName"
	srcLabel                    = "ssh.monitor.io/SourceIP"
	annotationByControllerKey   = "ssh.monitor.io/managedBy"
	annotationByControllerValue = "sshAuditController"
)

func generateAnnotations(entity *v1alpha1.SshCommandEntity) map[string]string {
	return map[string]string{annotationByControllerKey: annotationByControllerValue}
}

func generateCrLabels(entity *v1alpha1.SshCommandEntity) map[string]string {
	return map[string]string{nodeLabel: entity.Node, srcLabel: entity.SrcIp}
}
