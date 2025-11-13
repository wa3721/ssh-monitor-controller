package command

import (
	"context"
	"ssh-monitor-controller/api/v1alpha1"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

const (
	nodeLabel                   = "ssh.monitor.io/NodeName"
	srcLabel                    = "ssh.monitor.io/SourceIP"
	annotationByControllerKey   = "ssh.monitor.io/managedBy"
	annotationByControllerValue = "sshAuditController"
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
			err := processCommand(cmd, &cEntity)
			if err != nil {
				continue
			}
			//收到新的信号之后
			//先找到创建时间为当天，并且满足label上同时存在 node 和scrip同时相同的cr，如果没有创建一个新的，如果存在，更新存在的cr
			existCr, err := l.findExistCr(ctx, &cEntity)
			if err != nil {
				l.Log.Error(err, "Failed to find standard crs")
				continue
			}
			//更新
			if existCr != nil {
				err = l.updateExistCr(ctx, existCr.DeepCopy(), existCr, &cEntity)
				if err != nil {
					l.Log.Error(err, "Failed to update CR")
					l.EventRecorder.Eventf(existCr, corev1.EventTypeWarning, "UpdateFailed", "Failed to append command into this. cmd is %s,error is %v,will skip...", cEntity.Command, err)
					continue
				}
			} else {
				//创建
				err = l.createCr(ctx, &cEntity)
				if err != nil {
					l.Log.Error(err, "Failed to create new command")
					continue
				}
			}
		default:
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func (l *Listener) createCr(ctx context.Context, cEntity *v1alpha1.SshCommandEntity) error {
	o := func() error {
		//创建
		cr := newSshCommandCr(cEntity)
		err := l.Client.Create(ctx, cr)
		if err != nil {
			l.Log.Error(err, "Failed to create new command")
			return err
		}
		return nil
	}
	return retryWithDefault(ctx, o)
}

func (l *Listener) updateExistCr(ctx context.Context, cr, existCr *v1alpha1.SshCommandAudit, cEntity *v1alpha1.SshCommandEntity) error {
	o := func() error {
		patch := client.MergeFrom(cr)
		existCr.Spec.SshCommandEntity = append(existCr.Spec.SshCommandEntity, updateCommandEntity(cEntity))
		err := l.Client.Patch(ctx, existCr, patch)
		if err != nil {
			l.Log.Error(err, "Update failed for an existing CR.", "cmd", cEntity)
			return err
		}
		return nil
	}

	return retryWithDefault(ctx, o)

}

func (l *Listener) findExistCr(ctx context.Context, entity *v1alpha1.SshCommandEntity) (*v1alpha1.SshCommandAudit, error) {
	var maxRetries = 3
	var allCrs v1alpha1.SshCommandAuditList
	//列出所有打过标签的cr
	for i := 0; i < maxRetries; i++ {
		lastErr := l.Client.List(ctx, &allCrs, client.MatchingLabels{nodeLabel: entity.Node, srcLabel: entity.SrcIp})
		if lastErr == nil {
			break
		}
		l.Log.Error(lastErr, "Failed to list all labeled crs", "attempt", i+1, "maxRetries", maxRetries)
		if i == maxRetries-1 {
			return nil, lastErr
		}
		waitTime := time.Duration(1<<uint(i)) * time.Second
		if waitTime > 10*time.Second {
			waitTime = 10 * time.Second
		}
		select {
		case <-time.After(waitTime):
			// 继续重试
		case <-ctx.Done():
			l.Log.Info("Context cancelled, aborting retry")
			return nil, ctx.Err()
		}
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

func generateAnnotations() map[string]string {
	return map[string]string{annotationByControllerKey: annotationByControllerValue}
}

func generateCrLabels(entity *v1alpha1.SshCommandEntity) map[string]string {
	return map[string]string{nodeLabel: entity.Node, srcLabel: entity.SrcIp}
}

func (l *Listener) Start(ctx context.Context) error {
	l.Log.Info("starting listener for sshcommandaudit...")
	go l.ListenAndCreateOrUpdateCr(ctx)
	return nil
}
