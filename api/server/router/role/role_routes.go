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

package role

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/caoyingjunz/gopixiu/api/server/httpstatus"
	"github.com/caoyingjunz/gopixiu/api/server/httputils"
	"github.com/caoyingjunz/gopixiu/api/types"
	"github.com/caoyingjunz/gopixiu/pkg/db/model"
	"github.com/caoyingjunz/gopixiu/pkg/pixiu"
	"github.com/caoyingjunz/gopixiu/pkg/util"
)

// roles godoc
// @Summary      Create a role
// @Description  Create a role
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        data body types.RoleReq true "role info"
// @Success      200  {object}  httputils.HttpOK
// @Failure      400  {object}  httputils.HttpError
// @Router       /roles [post]
func (o *roleRouter) addRole(c *gin.Context) {
	r := httputils.NewResponse()
	var role types.RoleReq
	if err := c.ShouldBindJSON(&role); err != nil {
		httputils.SetFailed(c, r, httpstatus.ParamsError)
		return
	}

	exist := pixiu.CoreV1.Role().CheckRoleIsExist(c, role.Name)
	if exist {
		httputils.SetFailed(c, r, httpstatus.RoleExistError)
		return
	}

	if _, err := pixiu.CoreV1.Role().Create(c, &role); err != nil {
		httputils.SetFailed(c, r, httpstatus.OperateFailed)
		return
	}
	httputils.SetSuccess(c, r)
}

// roles godoc
// @Summary      update role
// @Description  update role
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "role ID"  Format(int64)
// @Param        data body types.RoleReq true "role info"
// @Success      200  {object}  httputils.HttpOK
// @Failure      400  {object}  httputils.HttpError
// @Router       /roles/{id} [put]
func (o *roleRouter) updateRole(c *gin.Context) {
	r := httputils.NewResponse()
	var role model.Role // TODO 后续优化
	if err := c.ShouldBindJSON(&role); err != nil {
		httputils.SetFailed(c, r, httpstatus.ParamsError)
		return
	}

	roleId, err := util.ParseInt64(c.Param("id"))
	if err != nil {
		httputils.SetFailed(c, r, httpstatus.ParamsError)
		return
	}

	_, err = pixiu.CoreV1.Role().Get(c, roleId)
	if err != nil {
		httputils.SetFailed(c, r, httpstatus.RoleNotExistError)
		return
	}

	if err = pixiu.CoreV1.Role().Update(c, &role, roleId); err != nil {
		httputils.SetFailed(c, r, httpstatus.OperateFailed)
		return
	}

	httputils.SetSuccess(c, r)
}

// roles godoc
// @Summary      delete role
// @Description  delete role
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "role ID"  Format(int64)
// @Success      200  {object}  httputils.HttpOK
// @Failure      400  {object}  httputils.HttpError
// @Router       /roles/{id} [delete]
func (o *roleRouter) deleteRole(c *gin.Context) {
	r := httputils.NewResponse()
	rid, err := util.ParseInt64(c.Param("id"))
	if err != nil {
		httputils.SetFailed(c, r, httpstatus.ParamsError)
		return
	}

	_, err = pixiu.CoreV1.Role().Get(c, rid)
	if err != nil {
		httputils.SetFailed(c, r, httpstatus.RoleNotExistError)
		return
	}

	if err = pixiu.CoreV1.Role().Delete(c, rid); err != nil {
		httputils.SetFailed(c, r, httpstatus.OperateFailed)
		return
	}

	httputils.SetSuccess(c, r)
}

// roles godoc
// @Summary      get role by role id
// @Description  get role by role id
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "role ID"  Format(int64)
// @Success      200  {object}  httputils.HttpOK
// @Failure      400  {object}  httputils.HttpError
// @Router       /roles/{id} [get]
func (o *roleRouter) getRole(c *gin.Context) {
	r := httputils.NewResponse()
	rid, err := util.ParseInt64(c.Param("id"))
	if err != nil {
		httputils.SetFailed(c, r, httpstatus.ParamsError)
		return
	}

	r.Result, err = pixiu.CoreV1.Role().Get(context.TODO(), rid)
	if err != nil {
		httputils.SetFailed(c, r, httpstatus.OperateFailed)
		return
	}

	httputils.SetSuccess(c, r)
}

// roles godoc
// @Summary      list roles
// @Description  list roles
// @Tags         roles
// @Accept       json
// @Produce      json
// @Success      200  {object}  httputils.Response{result=model.Role}
// @Failure      400  {object}  httputils.HttpError
// @Router       /roles [get]
func (o *roleRouter) listRoles(c *gin.Context) {
	r := httputils.NewResponse()
	var err error
	if r.Result, err = pixiu.CoreV1.Role().List(c); err != nil {
		httputils.SetFailed(c, r, httpstatus.OperateFailed)
		return
	}

	httputils.SetSuccess(c, r)
}

// roles godoc
// @Summary      get permissions by role id
// @Description  get permissions by role id
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "role ID"  Format(int64)
// @Success      200  {object}  httputils.Response{result=model.Menu}
// @Failure      400  {object}  httputils.HttpError
// @Router       /roles/{id}/menus [get]
func (o *roleRouter) getMenusByRole(c *gin.Context) {
	r := httputils.NewResponse()
	rid, err := util.ParseInt64(c.Param("id"))
	if err != nil {
		httputils.SetFailed(c, r, httpstatus.ParamsError)
		return
	}

	_, err = pixiu.CoreV1.Role().Get(c, rid)
	if err != nil {
		httputils.SetFailed(c, r, httpstatus.RoleNotExistError)
		return
	}

	if r.Result, err = pixiu.CoreV1.Role().GetMenusByRoleID(c, rid); err != nil {
		httputils.SetFailed(c, r, httpstatus.OperateFailed)
		return
	}
	httputils.SetSuccess(c, r)
}

// roles godoc
// @Summary      set permissions for role
// @Description  set permissions for role
// @Tags         roles
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "role ID"  Format(int64)
// @Param        data body types.Menus true "menu ids"
// @Success      200  {object}  httputils.Response{result=model.Menu}
// @Failure      400  {object}  httputils.HttpError
// @Router       /roles/{id}/menus [post]
func (o *roleRouter) setRoleMenus(c *gin.Context) {
	r := httputils.NewResponse()
	rid, err := util.ParseInt64(c.Param("id"))
	if err != nil {
		httputils.SetFailed(c, r, httpstatus.ParamsError)
		return
	}

	_, err = pixiu.CoreV1.Role().Get(c, rid)
	if err != nil {
		httputils.SetFailed(c, r, httpstatus.OperateFailed)
		return
	}

	var menuIds types.Menus
	if err = c.ShouldBindJSON(&menuIds); err != nil {
		httputils.SetFailed(c, r, httpstatus.ParamsError)
		return
	}

	if err = pixiu.CoreV1.Role().SetRole(c, rid, menuIds.MenuIDS); err != nil {
		httputils.SetFailed(c, r, httpstatus.OperateFailed)
		return
	}

	httputils.SetSuccess(c, r)
}
