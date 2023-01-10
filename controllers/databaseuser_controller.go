/*
Copyright 2023.

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
	"os"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	postgresqlv1alpha1 "external-database-operator/api/v1alpha1"
	"external-database-operator/postgres"
)

// DatabaseUserReconciler reconciles a DatabaseUser object
type DatabaseUserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const databaseUserFinalizer = "postgresql.my.domain/databaseUserFinalizer"

//+kubebuilder:rbac:groups=postgresql.my.domain,resources=databaseusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=postgresql.my.domain,resources=databaseusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=postgresql.my.domain,resources=databaseusers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DatabaseUser object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *DatabaseUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("DatabaseUserReconciler")

	//Parsing databaseUser object from request
	databaseUser := &postgresqlv1alpha1.DatabaseUser{}
	err := r.Client.Get(ctx, req.NamespacedName, databaseUser)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("DatabaseUser instance not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//Connecting to database
	pg, err := postgres.New(os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASS"), os.Getenv("DB_NAME"), logger)
	if err != nil {
		logger.Error(err, "Error connecting to database")
	}
	defer pg.CloseConnection()

	//Deletion logic
	if !databaseUser.GetDeletionTimestamp().IsZero() {
		err := pg.DropRole(databaseUser.Spec.Name)
		if err != nil {
			return ctrl.Result{}, err
		}
		//Remove finalizer
		controllerutil.RemoveFinalizer(databaseUser, databaseUserFinalizer)
		err = r.Update(ctx, databaseUser)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	//Creation logic
	err = pg.CreateRole(databaseUser.Spec.Name)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Add finalizer for this CRD
	if !controllerutil.ContainsFinalizer(databaseUser, databaseUserFinalizer) {
		controllerutil.AddFinalizer(databaseUser, databaseUserFinalizer)
		err = r.Update(ctx, databaseUser)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&postgresqlv1alpha1.DatabaseUser{}).
		Complete(r)
}
