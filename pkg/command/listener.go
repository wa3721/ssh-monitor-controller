package command

import (
	"github.com/go-logr/logr"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"ssh-monitor-controller/api/v1alpha1"
)

type Listener struct {
	client        client.Client
	scheme        *runtime.Scheme
	log           logr.Logger
	EventRecorder record.EventRecorder
}

func (l *Listener) ListenAndCreateCr() {
	log := l.log.WithName("ListenAndCreateCr")
	for {
		select {
		case cmd := <-CmdChannel:
			var cEntity v1alpha1.SshCommandEntity
			err := json.Unmarshal([]byte(*cmd), &cEntity)
			if err != nil {
				log.Error(err, "Failed to unmarshal command")
			}
			// create cr
		}
	}
}

func newSshCommandCr() *v1alpha1.SshCommandAudit {
	return &v1alpha1.SshCommandAudit{
		TypeMeta: v1.TypeMeta{
			Kind:       "SshCommandAudit",
			APIVersion: v1alpha1.GroupVersion.String(),
		},
		//ObjectMeta
	}
}
