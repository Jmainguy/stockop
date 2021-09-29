/*
Copyright 2021 Jonathan Mainguy.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	finnhub "github.com/Finnhub-Stock-API/finnhub-go/v2"
	"github.com/go-logr/logr"
	stockopv1 "github.com/jmainguy/stockop/api/v1"
	"os"
	"time"
)

// StockReconciler reconciles a Stock object
type StockReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=stockop.soh.re,resources=stocks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=stockop.soh.re,resources=stocks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=stockop.soh.re,resources=stocks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Stock object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *StockReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("stock", req.NamespacedName)

	// your logic here
	var stock stockopv1.Stock
	if err := r.Get(ctx, req.NamespacedName, &stock); err != nil {
		log.Error(err, "unable to fetch Stock")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cfg := finnhub.NewConfiguration()
	finnhubToken := os.Getenv("finnhubToken")
	cfg.AddDefaultHeader("X-Finnhub-Token", finnhubToken)
	finnhubClient := finnhub.NewAPIClient(cfg).DefaultApi

	stockName := strings.ToUpper(stock.Name)
	quote, _, err := finnhubClient.Quote(context.Background()).Symbol(stockName).Execute()
	if err != nil {
		log.Error(err, "unable to fetch current price", "stock", &stock)
		return ctrl.Result{}, err
	}
	currentPrice := fmt.Sprintf("%f", *quote.C)
	stock.Status.CurrentPrice = currentPrice
	err = r.Status().Update(ctx, &stock)
	if err != nil {
		log.Error(err, "unable to update status", "stock", &stock)
		return ctrl.Result{}, err
	}

	log.Info("Reconciliation complete")
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StockReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&stockopv1.Stock{}).
		Complete(r)
}
