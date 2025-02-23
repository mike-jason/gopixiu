/*
Copyright 2021 The Pixiu Authors.

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

package cloud

import (
	"context"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"

	"github.com/caoyingjunz/gopixiu/api/server/httputils"
	"github.com/caoyingjunz/gopixiu/api/types"
	"github.com/caoyingjunz/gopixiu/pkg/pixiu"
)

func (s *cloudRouter) createNamespace(c *gin.Context) {
	r := httputils.NewResponse()
	var (
		err         error
		listOptions types.CloudOptions
		namespace   corev1.Namespace
	)
	if err = c.ShouldBindUri(&listOptions); err != nil {
		httputils.SetFailed(c, r, err)
		return
	}
	if err = c.ShouldBindJSON(&namespace); err != nil {
		httputils.SetFailed(c, r, err)
		return
	}
	if err = pixiu.CoreV1.Cloud().Namespaces(listOptions.CloudName).Create(context.TODO(), namespace); err != nil {
		httputils.SetFailed(c, r, err)
		return
	}

	httputils.SetSuccess(c, r)
}

func (s *cloudRouter) updateNamespace(c *gin.Context) {
	r := httputils.NewResponse()
	var (
		err          error
		cloudOptions types.CloudOptions
		namespace    corev1.Namespace
	)
	if err = c.ShouldBindUri(&cloudOptions); err != nil {
		httputils.SetFailed(c, r, err)
		return
	}
	if err = c.ShouldBindJSON(&namespace); err != nil {
		httputils.SetFailed(c, r, err)
		return
	}
	r.Result, err = pixiu.CoreV1.Cloud().Namespaces(cloudOptions.CloudName).Update(context.TODO(), namespace)
	if err != nil {
		httputils.SetFailed(c, r, err)
		return
	}
	httputils.SetSuccess(c, r)
}

func (s *cloudRouter) deleteNamespace(c *gin.Context) {
	r := httputils.NewResponse()
	var (
		err              error
		namespaceOptions types.NamespaceOptions
	)
	if err = c.ShouldBindUri(&namespaceOptions); err != nil {
		httputils.SetFailed(c, r, err)
		return
	}
	if err = pixiu.CoreV1.Cloud().Namespaces(namespaceOptions.CloudName).Delete(context.TODO(), namespaceOptions.ObjectName); err != nil {
		httputils.SetFailed(c, r, err)
		return
	}

	httputils.SetSuccess(c, r)
}

func (s *cloudRouter) getNamespace(c *gin.Context) {
	r := httputils.NewResponse()
	var (
		err              error
		namespaceOptions types.NamespaceOptions
	)
	if err = c.ShouldBindUri(&namespaceOptions); err != nil {
		httputils.SetFailed(c, r, err)
	}
	r.Result, err = pixiu.CoreV1.Cloud().Namespaces(namespaceOptions.CloudName).Get(context.TODO(), namespaceOptions.ObjectName)
	if err != nil {
		httputils.SetFailed(c, r, err)
		return
	}
	httputils.SetSuccess(c, r)
}

func (s *cloudRouter) listNamespaces(c *gin.Context) {
	r := httputils.NewResponse()
	var (
		err          error
		cloudOptions types.CloudOptions
	)
	if err = c.ShouldBindUri(&cloudOptions); err != nil {
		httputils.SetFailed(c, r, err)
		return
	}
	if r.Result, err = pixiu.CoreV1.Cloud().Namespaces(cloudOptions.CloudName).List(context.TODO()); err != nil {
		httputils.SetFailed(c, r, err)
		return
	}

	httputils.SetSuccess(c, r)
}
