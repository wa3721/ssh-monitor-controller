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
	Client        client.Client
	Scheme        *runtime.Scheme
	Log           logr.Logger
	EventRecorder record.EventRecorder
}

func (l *Listener) ListenAndCreateOrUpdateCr(ctx context.Context) {
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
			todayCr, err := l.findLabeledCrsToday(ctx, &cEntity)
			if err != nil {
				commandListenerLog.Error(err, "Failed to find standard crs")
			}
			//更新
			if todayCr != nil {
				patch := client.MergeFrom(todayCr.DeepCopy())
				todayCr.Spec.SshCommandEntity = append(todayCr.Spec.SshCommandEntity, updateCommandEntity(&cEntity))
				err = l.Client.Patch(ctx, todayCr, patch)
				if err != nil {
					commandListenerLog.Error(err, "Failed to patch new command")
				}
			} else {
				//创建
				cr := newSshCommandCr(&cEntity)
				err = l.Client.Create(ctx, cr)
				if err != nil {
					commandListenerLog.Error(err, "Failed to create new command")
				}
			}
		default:
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func (l *Listener) findLabeledCrsToday(ctx context.Context, entity *v1alpha1.SshCommandEntity) (*v1alpha1.SshCommandAudit, error) {
	var allCrs v1alpha1.SshCommandAuditList
	//列出所有打过标签的cr
	err := l.Client.List(ctx, &allCrs, client.MatchingLabels{nodeLabel: entity.Node, srcLabel: entity.SrcIp})
	if err != nil {
		commandListenerLog.Error(err, "Failed to list all labeled crs")
		return nil, err
	}
	// 长度为0说明没有创建过cr
	if len(allCrs.Items) == 0 {
		return nil, nil
	}
	//判断是否存在定义的annotation
	//判断是否是今天
	for _, cr := range allCrs.Items {
		if cr.CreationTimestamp.Time.Format(time.DateOnly) == time.Now().Format(time.DateOnly) {
			//今天创建的
			if cr.Annotations[annotationByControllerKey] == annotationByControllerValue {
				return &cr, nil
			}

		}
	}
	return nil, nil

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
			Annotations:  generateAnnotations(),
			Labels:       generateCrLabels(entity),
		},
		Spec: v1alpha1.SshCommandAuditSpec{
			SshCommandEntity: []*v1alpha1.SshCommandEntity{
				{
					TimeStamp: entity.TimeStamp,
					User:      entity.User,
					SrcPort:   entity.SrcPort,
					WorkDir:   entity.WorkDir,
					Command:   entity.Command,
					ExitCode:  entity.ExitCode,
				},
			},
		},
	}
}

func updateCommandEntity(entity *v1alpha1.SshCommandEntity) *v1alpha1.SshCommandEntity {
	return &v1alpha1.SshCommandEntity{
		TimeStamp: entity.TimeStamp,
		User:      entity.User,
		SrcPort:   entity.SrcPort,
		WorkDir:   entity.WorkDir,
		Command:   entity.Command,
		ExitCode:  entity.ExitCode,
	}
}

const (
	nodeLabel                   = "ssh.monitor.io/NodeName"
	srcLabel                    = "ssh.monitor.io/SourceIP"
	annotationByControllerKey   = "ssh.monitor.io/managedBy"
	annotationByControllerValue = "sshAuditController"
)

func generateAnnotations() map[string]string {
	return map[string]string{annotationByControllerKey: annotationByControllerValue}
}

func generateCrLabels(entity *v1alpha1.SshCommandEntity) map[string]string {
	return map[string]string{nodeLabel: entity.Node, srcLabel: entity.SrcIp}
}

func (l *Listener) Start(ctx context.Context) error {
	go l.ListenAndCreateOrUpdateCr(ctx)
	return nil
}
