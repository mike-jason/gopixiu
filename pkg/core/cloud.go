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

package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"github.com/caoyingjunz/gopixiu/api/types"
	"github.com/caoyingjunz/gopixiu/pkg/core/client"
	pixiukubernetes "github.com/caoyingjunz/gopixiu/pkg/core/kubernetes"
	"github.com/caoyingjunz/gopixiu/pkg/db"
	"github.com/caoyingjunz/gopixiu/pkg/db/model"
	"github.com/caoyingjunz/gopixiu/pkg/log"
	typesv2 "github.com/caoyingjunz/gopixiu/pkg/types"
	"github.com/caoyingjunz/gopixiu/pkg/util/cipher"
	"github.com/caoyingjunz/gopixiu/pkg/util/uuid"
)

var clientError = fmt.Errorf("failed to found clout client")

type CloudGetter interface {
	Cloud() CloudInterface
}

type CloudInterface interface {
	Create(ctx context.Context, obj *types.Cloud) error
	Update(ctx context.Context, obj *types.Cloud) error
	Delete(ctx context.Context, cid int64) error
	Get(ctx context.Context, cid int64) (*types.Cloud, error)
	List(ctx context.Context, paging *types.PageOptions) (interface{}, error)
	Build(ctx context.Context, obj *types.BuildCloud) error

	// Ping 检查 kubeConfig 与 kubernetes 集群的连通状态
	Ping(ctx context.Context, kubeConfigData []byte) error
	// Load 加载已经存在的 cloud 客户端
	Load(stopCh chan struct{}) error

	// kubernetes 资源的接口定义
	pixiukubernetes.NamespacesGetter
	pixiukubernetes.ServicesGetter
	pixiukubernetes.StatefulSetGetter
	pixiukubernetes.DeploymentsGetter
	pixiukubernetes.DaemonSetGetter
	pixiukubernetes.JobsGetter
	pixiukubernetes.IngressGetter
	pixiukubernetes.EventsGetter
	pixiukubernetes.NodesGetter
	pixiukubernetes.PodsGetter
	pixiukubernetes.KubeConfigGetter
}

const (
	page     = 1
	pageSize = 10
)

var clientSets client.ClientsInterface

type cloud struct {
	app     *pixiu
	factory db.ShareDaoFactory
}

func newCloud(c *pixiu) CloudInterface {
	return &cloud{
		app:     c,
		factory: c.factory,
	}
}

func (c *cloud) preCreate(ctx context.Context, obj *types.Cloud) error {
	if len(obj.Name) == 0 {
		return fmt.Errorf("invalid empty cloud name")
	}
	if len(obj.KubeConfig) == 0 {
		return fmt.Errorf("invalid empty kubeconfig data")
	}
	// 集群类型支持 自建和标准，默认为标准

	// TODO: 其他规范创建前检查

	return nil
}

func (c *cloud) Create(ctx context.Context, obj *types.Cloud) error {
	if err := c.preCreate(ctx, obj); err != nil {
		log.Logger.Errorf("failed to pre-check for %s created: %v", obj.Name, err)
		return err
	}
	// 直接对 kubeConfig 内容进行加密
	encryptData, err := cipher.Encrypt(obj.KubeConfig)
	if err != nil {
		log.Logger.Errorf("failed to encrypt kubeConfig: %v", err)
		return err
	}

	// 先构造 clientSet，如果有异常，直接返回
	clientSet, err := c.newClientSet(obj.KubeConfig)
	if err != nil {
		log.Logger.Errorf("failed to create %s clientSet: %v", obj.Name, err)
		return err
	}
	// 获取 k8s 集群信息: k8s 版本，节点数量，资源信息
	nodes, err := clientSet.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil || len(nodes.Items) == 0 {
		log.Logger.Errorf("failed to connected to k8s cluster: %v", err)
		return err
	}

	var (
		kubeVersion string
		resources   string
	)
	// 第一个节点的版本作为集群版本
	node := nodes.Items[0]
	nodeStatus := node.Status
	kubeVersion = nodeStatus.NodeInfo.KubeletVersion

	// TODO: 未处理 resources
	if _, err = c.factory.Cloud().Create(ctx, &model.Cloud{
		Name:        "atm-" + uuid.NewUUID()[:8],
		AliasName:   obj.Name,
		CloudType:   typesv2.StandardCloud,
		KubeVersion: kubeVersion,
		KubeConfig:  encryptData,
		NodeNumber:  len(nodes.Items),
		Resources:   resources,
	}); err != nil {
		log.Logger.Errorf("failed to create %s cloud: %v", obj.Name, err)
		return err
	}

	// TODO: 根据传参确定是否创建默认ns
	// 创建 pixiu-system 命名空间，用于安装内置的控制器
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: typesv2.PixiuNamespace,
		},
	}
	_, err = clientSet.CoreV1().Namespaces().Create(context.Background(), namespace, metav1.CreateOptions{})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		log.Logger.Errorf("create default namespace error: %v", err)
		return err
	}

	clientSets.Add(obj.Name, clientSet)
	return nil
}

func (c *cloud) preBuild(ctx context.Context, obj *types.BuildCloud) error {
	return nil
}

// Build 构造 Cloud
func (c *cloud) Build(ctx context.Context, obj *types.BuildCloud) error {
	if err := c.preBuild(ctx, obj); err != nil {
		log.Logger.Errorf("failed to pre-check for %s build: %v", obj.Name, err)
		return err
	}

	// step1: 创建 cloud
	cloudObj, err := c.factory.Cloud().Create(ctx, &model.Cloud{
		Name:        "pix-" + uuid.NewUUID()[:8],
		AliasName:   obj.AliasName,
		Status:      typesv2.InitializeStatus, // 初始化状态
		CloudType:   typesv2.BuildCloud,
		KubeVersion: obj.Kubernetes.Version,
		Description: obj.Description,
	})
	if err != nil {
		log.Logger.Errorf("failed to create cloud %s: %v", obj.AliasName, err)
		return err
	}
	cid := cloudObj.Id

	// step2: 创建 k8s cluster
	if err = c.buildCluster(ctx, cid, obj.Kubernetes); err != nil {
		log.Logger.Errorf("failed to build %s cloud cluster: %v", obj.AliasName, err)
		_ = c.forceDelete(ctx, cid)
		return err
	}

	// step3: 创建 nodes
	if err = c.buildNodes(ctx, cid, obj.Kubernetes); err != nil {
		log.Logger.Errorf("failed to build %s cloud nodes: %v", obj.AliasName, err)
		_ = c.forceDelete(ctx, cid)
		return err
	}

	// 立刻进行部署
	if obj.Immediate {
		go c.StartDeployCluster(ctx, cid)
	}
	return nil
}

func (c *cloud) buildCluster(ctx context.Context, cid int64, kubeObj *types.KubernetesSpec) error {
	if err := c.factory.Cloud().CreateCluster(ctx, &model.Cluster{
		CloudId:     cid,
		ApiServer:   kubeObj.ApiServer,
		Version:     kubeObj.Version,
		Runtime:     kubeObj.Runtime,
		Cni:         kubeObj.Cni,
		ServiceCidr: kubeObj.ServiceCidr,
		PodCidr:     kubeObj.PodCidr,
		ProxyMode:   kubeObj.ProxyMode,
	}); err != nil {
		log.Logger.Errorf("failed to create cluster: %v", err)
		return err
	}

	return nil
}

func (c *cloud) buildNodes(ctx context.Context, cid int64, kubeObj *types.KubernetesSpec) error {
	var kubeNodes []model.Node
	// master 节点
	for _, master := range kubeObj.Masters {
		kubeNodes = append(kubeNodes, model.Node{
			CloudId:  cid,
			Role:     typesv2.MasterRole,
			HostName: master.HostName,
			Address:  master.Address,
			User:     master.User,
			Password: master.Password,
		})
	}
	// node 节点
	for _, node := range kubeObj.Nodes {
		kubeNodes = append(kubeNodes, model.Node{
			CloudId:  cid,
			Role:     typesv2.NodeRole,
			HostName: node.HostName,
			Address:  node.Address,
			User:     node.User,
			Password: node.Password,
		})
	}
	if err := c.factory.Cloud().CreateNodes(ctx, kubeNodes); err != nil {
		log.Logger.Errorf("failed to create %d nodes: %v", cid, err)
		return err
	}

	return nil
}

// 回滚 TODO
func (c *cloud) forceDelete(ctx context.Context, cid int64) error {
	return nil
}

func (c *cloud) Update(ctx context.Context, obj *types.Cloud) error { return nil }

// Delete 删除 cloud
// 同时根据 cloud 的类型，清除级联资源，标准资源直接删除，自建资源删除 cluster 和 nodes
func (c *cloud) Delete(ctx context.Context, cid int64) error {
	obj, err := c.factory.Cloud().Delete(ctx, cid)
	if err != nil {
		log.Logger.Errorf("failed to delete %s cloud: %v", cid, err)
		return err
	}
	clientSets.Delete(obj.Name)

	// 目前，仅自建的k8s集群需要清理下属资源，下属资源有 cluster 和 nodes
	if obj.CloudType == typesv2.BuildCloud {
		_ = c.internalDelete(ctx, cid)
	}
	return nil
}

// TODO: 可以并发操作
// 清理 cloud 下属资源
func (c *cloud) internalDelete(ctx context.Context, cid int64) error {
	if err := c.factory.Cloud().DeleteCluster(ctx, cid); err != nil {
		return err
	}
	if err := c.factory.Cloud().DeleteNodes(ctx, cid); err != nil {
		return err
	}

	return nil
}

func (c *cloud) Get(ctx context.Context, cid int64) (*types.Cloud, error) {
	cloudObj, err := c.factory.Cloud().Get(ctx, cid)
	if err != nil {
		log.Logger.Errorf("failed to get %d cloud: %v", cid, err)
		return nil, err
	}

	return c.model2Type(cloudObj), nil
}

// List 返回全量查询或者分页查询
func (c *cloud) List(ctx context.Context, pageOption *types.PageOptions) (interface{}, error) {
	var cs []types.Cloud

	// 根据是否有 Page 判断是全量查询还是分页查询，如果没有则判定为全量查询，如果有，判定为分页查询
	// TODO: 判断方法略显粗糙，后续优化
	// 分页查询
	if pageOption.Page != 0 {
		if pageOption.Page < 0 {
			pageOption.Page = page
		}
		if pageOption.Limit <= 0 {
			pageOption.Limit = pageSize
		}
		cloudObjs, total, err := c.factory.Cloud().PageList(ctx, pageOption.Page, pageOption.Limit)
		if err != nil {
			log.Logger.Errorf("failed to page %d limit %d list  clouds: %v", pageOption.Page, pageOption.Limit, err)
			return nil, err
		}
		// 类型转换
		for _, cloudObj := range cloudObjs {
			cs = append(cs, *c.model2Type(&cloudObj))
		}

		pageClouds := make(map[string]interface{})
		pageClouds["data"] = cs
		pageClouds["total"] = total
		return pageClouds, nil
	}

	// 全量查询
	cloudObjs, err := c.factory.Cloud().List(ctx)
	if err != nil {
		log.Logger.Errorf("failed to list clouds: %v", err)
		return nil, err
	}
	for _, cloudObj := range cloudObjs {
		cs = append(cs, *c.model2Type(&cloudObj))
	}

	return cs, nil
}

func (c *cloud) Ping(ctx context.Context, kubeConfigData []byte) error {
	// 先构造 clientSet，如果有异常，直接返回
	clientSet, err := c.newClientSet(kubeConfigData)
	if err != nil {
		log.Logger.Errorf("failed to create clientSet: %v", err)
		return err
	}
	_, err = clientSet.CoreV1().Namespaces().Get(ctx, "kube-system", metav1.GetOptions{})
	if err != nil {
		return nil
	}

	return nil
}

func (c *cloud) Load(stopCh chan struct{}) error {
	// 初始化云客户端
	clientSets = client.NewCloudClients()

	cloudObjs, err := c.factory.Cloud().List(context.TODO())
	if err != nil {
		log.Logger.Errorf("failed to list exist clouds: %v", err)
		return err
	}
	for _, cloudObj := range cloudObjs {
		// TODO: 仅加载状态正常的集群，异常的加入到异常列表
		if cloudObj.Status != 0 {
			continue
		}
		kubeConfig, err := cipher.Decrypt(cloudObj.KubeConfig)
		if err != nil {
			return err
		}
		clientSet, err := c.newClientSet(kubeConfig)
		if err != nil {
			log.Logger.Errorf("failed to create %s clientSet: %v", cloudObj.Name, err)
			return err
		}
		clientSets.Add(cloudObj.Name, clientSet)
		klog.V(2).Infof("load cloud %s success", cloudObj.Name)
	}

	// 启动集群检查状态检查
	go c.ClusterHealthCheck(stopCh)
	return nil
}

func (c *cloud) ClusterHealthCheck(stopCh chan struct{}) {
	klog.V(2).Infof("starting cluster health check")
	// 存储旧的检查状态
	status := make(map[string]int)

	interval := time.Second * 5
	for {
		select {
		case <-time.After(interval):
			// 定时刷新status map
			for name := range status {
				if clientSets.Get(name) == nil {
					delete(status, name)
				}
			}
			// 定时检查cluster集群状态
			for name, cs := range clientSets.List() {
				var newStatus int
				var timeoutSeconds int64 = 2
				if _, err := cs.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{TimeoutSeconds: &timeoutSeconds, Limit: 1}); err != nil {
					log.Logger.Errorf("failed to check %s cluster: %v", name, err)
					newStatus = 1
				}
				// 对比状态是否改变
				if status[name] != newStatus {
					status[name] = newStatus
					_ = c.factory.Cloud().SetStatus(context.TODO(), name, newStatus)
				}
			}
		case <-stopCh:
			klog.Infof("shutting cluster health check")
			return
		}
	}
}

func (c *cloud) newClientSet(data []byte) (*kubernetes.Clientset, error) {
	kubeConfig, err := clientcmd.RESTConfigFromKubeConfig(data)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(kubeConfig)
}

func (c *cloud) model2Type(obj *model.Cloud) *types.Cloud {
	return &types.Cloud{
		IdMeta: types.IdMeta{
			Id:              obj.Id,
			ResourceVersion: obj.ResourceVersion,
		},
		Name:        obj.Name,
		AliasName:   obj.AliasName,
		Status:      obj.Status,
		CloudType:   obj.CloudType,
		KubeVersion: obj.KubeVersion,
		NodeNumber:  obj.NodeNumber,
		Resources:   obj.Resources,
		Description: obj.Description,
		TimeOption:  types.NewTypeTime(obj.GmtCreate, obj.GmtModified),
	}
}

// StartDeployCluster TODO
func (c *cloud) StartDeployCluster(ctx context.Context, cid int64) error {
	return nil
}
