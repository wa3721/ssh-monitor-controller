package controller

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"ssh-monitor-controller/api/v1alpha1"
	"strconv"
	"time"
)

// 定义规则配置

type Rule struct {
	Params map[string]string
}

// Collector 控制器结构体
type Collector struct {
	client.Client // 通过 client.Client 与 API Server 交互
	Log           logr.Logger
	RulesConf     *Rule
}

// 实现 Reconcile 逻辑

func (r *Collector) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("collector-controller evaluating CRs")

	// 获取所有 SshCommandAudit CR
	var crList v1alpha1.SshCommandAuditList
	if err := r.List(ctx, &crList); err != nil {
		r.Log.Error(err, "Failed to retrieve CR list")
		return ctrl.Result{}, err
	}

	var minRequeueAfter time.Duration // 用于计算最近的过期时间
	hasErrors := false

	for _, cr := range crList.Items {
		// 检查并计算是否需要删除此 CR
		result, err := r.evaluateCR(ctx, &cr)
		if err != nil {
			r.Log.Error(err, "Evaluation of CR rules failed", "name", cr.Name)
			hasErrors = true
			continue
		}
		// 更新最近的 requeue 时间
		if result.RequeueAfter > 0 && (minRequeueAfter == 0 || result.RequeueAfter < minRequeueAfter) {
			minRequeueAfter = result.RequeueAfter
		}
	}

	if hasErrors {
		return ctrl.Result{}, fmt.Errorf("an error occurred while processing part of the CR")
	}

	// 如果找到了需要重新协调的时间，则返回该时间
	if minRequeueAfter > 0 {
		return ctrl.Result{RequeueAfter: minRequeueAfter}, nil
	}

	// 默认 5 分钟后再进行全量扫描
	return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
}

func (r *Collector) evaluateCR(ctx context.Context, cr *v1alpha1.SshCommandAudit) (ctrl.Result, error) {
	// 计算 CR 的年龄
	age := time.Since(cr.CreationTimestamp.Time)
	retentionDays, _ := strconv.Atoi(r.RulesConf.Params["retentionDays"])
	maxAge := time.Duration(retentionDays) * 24 * time.Hour

	// 如果超过保留期限，则删除
	if age > maxAge {
		r.Log.Info("Delete expired CRs", "name", cr.Name, "age", age)
		if err := r.Delete(ctx, cr); err != nil {
			return ctrl.Result{RequeueAfter: 60 * time.Hour}, err
		}
		return ctrl.Result{}, nil
	}

	// 计算距离过期还有多久，并设置在此时间后重新协调
	timeUntilExpiration := maxAge - age
	return ctrl.Result{RequeueAfter: timeUntilExpiration}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Collector) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.SshCommandAudit{}).
		WithEventFilter(predicate.Funcs{
			//create cr
			CreateFunc: func(e event.CreateEvent) bool {
				return true
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return false
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
			GenericFunc: func(e event.GenericEvent) bool {
				return false
			},
		}).
		Named("collector").Complete(r)
}
