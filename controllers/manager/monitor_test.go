/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2023 Wind River Systems, Inc. */

package manager

import (
	"errors"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/starlingx/nfv/v1/systemconfigupdate"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Monitor", func() {
	Describe("Check return for monitorStrategyState", func() {
		Context("Fail to obtain vim client", func() {
			It("should return false", func() {
				dm := &Dummymanager{vimClientAvailable: false, gcShow: ""}
				got := monitorStrategyState(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyDeleted).To(BeFalse())
			})
		})
		Context("Fail to obtain strategy status", func() {
			It("should return false", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: ""}
				got := monitorStrategyState(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyDeleted).To(BeFalse())
			})
		})
		Context("Status is strategy applied", func() {
			It("should return false", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyReadyToApply}
				got := monitorStrategyState(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyActionSend).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeFalse())
			})
		})
		Context("Strategy apply error but before retry exceeds", func() {
			It("should return false and strategy action not sent", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyReadyToApply, strategyActionError: true, retryCount: 10}
				got := monitorStrategyState(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyActionSend).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeFalse())
			})
		})
		Context("Strategy apply error but retry exceeds", func() {
			It("should return true, strategy action not sent and strategy deleted", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyReadyToApply, strategyActionError: true, retryCount: DefaultMaxStrategyRetryCount + 1}
				got := monitorStrategyState(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyActionSend).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeTrue())
			})
		})
		Context("Status is build failed", func() {
			It("should return true and strategy deleted", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyBuildFailed}
				got := monitorStrategyState(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeTrue())
			})
		})
		Context("Status is strategy apply failed", func() {
			It("should return true and strategy deleted", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyApplyFailed}
				got := monitorStrategyState(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeTrue())
			})
		})
		Context("Status is strategy applying", func() {
			It("should return false", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyApplying}
				got := monitorStrategyState(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyDeleted).To(BeFalse())
			})
		})
		Context("Status is strategy apply failed", func() {
			It("should return true and strategy deleted", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyBuildTimeout}
				got := monitorStrategyState(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeTrue())
			})
		})
		Context("Status is strategy apply timeout", func() {
			It("should return true and strategy deleted", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyApplyTimeout}
				got := monitorStrategyState(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeTrue())
			})
		})
		Context("Status is strategy abort failed", func() {
			It("should return true and strategy deleted", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyAbortFailed}
				got := monitorStrategyState(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeTrue())
			})
		})
		Context("Status is strategy aborting", func() {
			It("should return false", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyAborting}
				got := monitorStrategyState(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyDeleted).To(BeFalse())
			})
		})
		Context("Status is strategy abort timeout", func() {
			It("should return true and strategy deleted", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyAbortTimeout}
				got := monitorStrategyState(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeTrue())
			})
		})
		Context("Status is strategy applied", func() {
			It("should return true and strategy deleted", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyApplied}
				got := monitorStrategyState(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeTrue())
			})
		})
	})

	Describe("Check return for ManageStrategy", func() {
		Context("Monitor version is not match to config version", func() {
			It("should return false", func() {
				dm := &Dummymanager{config_version: 0, monitor_version: 1}
				got := ManageStrategy(dm)
				Expect(got).To(BeFalse())
			})
		})
		Context("Reconcile is not finished", func() {
			It("should return false and strategy not created", func() {
				rsc := map[string]*ResourceInfo{
					"controller-0": {
						StrategyRequired: StrategyNotRequired,
						Reconciled:       false,
					},
				}
				dm := &Dummymanager{strategySent: false, resource: rsc}
				got := ManageStrategy(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyCreated).To(BeFalse())
			})
		})
		Context("Reconcile is finished but no strategy required", func() {
			It("should return false and strategy not created", func() {
				rsc := map[string]*ResourceInfo{
					"controller-0": {
						StrategyRequired: StrategyNotRequired,
						Reconciled:       true,
					},
				}
				dm := &Dummymanager{strategySent: false, resource: rsc}
				got := ManageStrategy(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyCreated).To(BeFalse())
			})
		})
		Context("Lock required for system", func() {
			It("should return false and strategy created and sent", func() {
				rsc := map[string]*ResourceInfo{
					"controller-0": {
						ResourceType:     ResourceSystem,
						StrategyRequired: StrategyLockRequired,
						Reconciled:       true,
					},
				}
				dm := &Dummymanager{strategySent: false, resource: rsc, vimClientAvailable: true}
				got := ManageStrategy(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyCreated).To(BeTrue())
				Expect(dm.strategySent).To(BeTrue())
				Expect(dm.strategyCreateRequest.ControllerApplyType).To(Equal("serial"))
				Expect(dm.strategyCreateRequest.WorkerApplyType).To(Equal("parallel"))
				Expect(dm.strategyCreateRequest.StorageApplyType).To(Equal("ignore"))
			})
		})
		Context("Lock required for controller", func() {
			It("should return false and strategy created and sent", func() {
				rsc := map[string]*ResourceInfo{
					"controller-0": {
						ResourceType:     ResourceHost,
						Personality:      PersonalityController,
						StrategyRequired: StrategyLockRequired,
						Reconciled:       true,
					},
				}
				dm := &Dummymanager{strategySent: false, resource: rsc, vimClientAvailable: true}
				got := ManageStrategy(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyCreated).To(BeTrue())
				Expect(dm.strategySent).To(BeTrue())
				Expect(dm.strategyCreateRequest.ControllerApplyType).To(Equal("serial"))
				Expect(dm.strategyCreateRequest.WorkerApplyType).To(Equal("parallel"))
				Expect(dm.strategyCreateRequest.StorageApplyType).To(Equal("ignore"))
			})
		})
		Context("Lock required for worker", func() {
			It("should return false and strategy created and sent", func() {
				rsc := map[string]*ResourceInfo{
					"controller-0": {
						ResourceType:     ResourceHost,
						Personality:      PersonalityWorker,
						StrategyRequired: StrategyLockRequired,
						Reconciled:       true,
					},
				}
				dm := &Dummymanager{strategySent: false, resource: rsc, vimClientAvailable: true}
				got := ManageStrategy(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyCreated).To(BeTrue())
				Expect(dm.strategySent).To(BeTrue())
				Expect(dm.strategyCreateRequest.ControllerApplyType).To(Equal("ignore"))
				Expect(dm.strategyCreateRequest.WorkerApplyType).To(Equal("parallel"))
				Expect(dm.strategyCreateRequest.StorageApplyType).To(Equal("ignore"))
			})
		})
		Context("Lock required for storage", func() {
			It("should return false and strategy created and sent", func() {
				rsc := map[string]*ResourceInfo{
					"controller-0": {
						ResourceType:     ResourceHost,
						Personality:      PersonalityStorage,
						StrategyRequired: StrategyLockRequired,
						Reconciled:       true,
					},
				}
				dm := &Dummymanager{strategySent: false, resource: rsc, vimClientAvailable: true}
				got := ManageStrategy(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyCreated).To(BeTrue())
				Expect(dm.strategySent).To(BeTrue())
				Expect(dm.strategyCreateRequest.ControllerApplyType).To(Equal("ignore"))
				Expect(dm.strategyCreateRequest.WorkerApplyType).To(Equal("ignore"))
				Expect(dm.strategyCreateRequest.StorageApplyType).To(Equal("serial"))
			})
		})
		Context("Strategy create error but before retry exceeds", func() {
			It("should return false and strategy not created", func() {
				rsc := map[string]*ResourceInfo{
					"controller-0": {
						ResourceType:     ResourceHost,
						Personality:      PersonalityController,
						StrategyRequired: StrategyLockRequired,
						Reconciled:       true,
					},
				}
				dm := &Dummymanager{strategySent: false, resource: rsc, vimClientAvailable: true, strategyCreateError: true, retryCount: 10}
				got := ManageStrategy(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyCreated).To(BeFalse())
			})
		})
		Context("Strategy create error but retry exceeds", func() {
			It("should return false and strategy not created", func() {
				rsc := map[string]*ResourceInfo{
					"controller-0": {
						ResourceType:     ResourceHost,
						Personality:      PersonalityController,
						StrategyRequired: StrategyLockRequired,
						Reconciled:       true,
					},
				}
				dm := &Dummymanager{strategySent: false, resource: rsc, vimClientAvailable: true, strategyCreateError: true, retryCount: DefaultMaxStrategyRetryCount + 1}
				got := ManageStrategy(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyCreated).To(BeFalse())
			})
		})
		Context("Strategy applying after strategy sent", func() {
			It("should return false", func() {
				dm := &Dummymanager{strategySent: true, vimClientAvailable: true, gcShow: StrategyApplying}
				got := ManageStrategy(dm)
				Expect(got).To(BeFalse())
			})
		})
		Context("Strategy applied after strategy sent", func() {
			It("should return true", func() {
				dm := &Dummymanager{strategySent: true, vimClientAvailable: true, gcShow: StrategyApplied}
				got := ManageStrategy(dm)
				Expect(got).To(BeTrue())
			})
		})
	})
})

// Dummy Manager
type Dummymanager struct {
	vimClientAvailable    bool
	gcShow                string
	strategyCreated       bool
	strategyCreateError   bool
	strategySent          bool
	strategyDeleted       bool
	strategyActionSend    bool
	strategyActionError   bool
	config_version        int
	monitor_version       int
	resource              map[string]*ResourceInfo
	strategyCreateRequest systemconfigupdate.SystemConfigUpdateOpts
	retryCount            int
}

func (m *Dummymanager) ResetPlatformClient(namespace string) error {
	return nil
}
func (m *Dummymanager) GetPlatformClient(namespace string) *gophercloud.ServiceClient {
	c := &gophercloud.ServiceClient{}
	return c
}
func (m *Dummymanager) GetKubernetesClient() client.Client {
	return nil
}
func (m *Dummymanager) BuildPlatformClient(namespace string, endpointName string, endpointType string) (*gophercloud.ServiceClient, error) {
	c := &gophercloud.ServiceClient{}
	return c, nil
}
func (m *Dummymanager) NotifySystemDependencies(namespace string) error {
	return nil
}
func (m *Dummymanager) NotifyResource(object client.Object) error {
	return nil
}
func (m *Dummymanager) SetSystemReady(namespace string, value bool) {

}
func (m *Dummymanager) GetSystemReady(namespace string) bool {
	return true
}
func (m *Dummymanager) SetSystemType(namespace string, value SystemType) {

}
func (m *Dummymanager) GetSystemType(namespace string) SystemType {
	return ""
}
func (m *Dummymanager) StartMonitor(monitor *Monitor, message string) error {
	return nil
}
func (m *Dummymanager) CancelMonitor(object client.Object) {

}
func (m *Dummymanager) SetResourceInfo(resourcetype string, personality string, resourcename string, reconciled bool, required string) {

}
func (m *Dummymanager) GetStrategyRequiredList() map[string]*ResourceInfo {
	return m.resource
}
func (m *Dummymanager) ListStrategyRequired() string {
	return ""
}
func (m *Dummymanager) UpdateConfigVersion() {

}
func (m *Dummymanager) GetConfigVersion() int {
	return m.config_version
}
func (m *Dummymanager) GetMonitorVersion() int {
	return m.monitor_version
}
func (m *Dummymanager) SetMonitorVersion(i int) {

}
func (m *Dummymanager) StrageySent() {
	m.strategySent = true
}
func (m *Dummymanager) GetStrageySent() bool {
	return m.strategySent
}
func (m *Dummymanager) ClearStragey() {

}
func (m *Dummymanager) GetStrageyNamespace() string {
	return ""
}
func (m *Dummymanager) GetVimClient() *gophercloud.ServiceClient {
	if m.vimClientAvailable {
		c := &gophercloud.ServiceClient{}
		return c
	} else {
		return nil
	}
}
func (m *Dummymanager) SetStrategyAppliedSent(namespace string, applied bool) error {
	return nil
}
func (m *Dummymanager) StartStrategyMonitor() {

}
func (m *Dummymanager) SetStrategyRetryCount(c int) error {
	return nil
}
func (m *Dummymanager) GetStrategyRetryCount() (int, error) {
	return m.retryCount, nil
}

func (m *Dummymanager) GcShow(c *gophercloud.ServiceClient) (*systemconfigupdate.SystemConfigUpdate, error) {
	if len(m.gcShow) == 0 {
		err := errors.New("test: no info available")
		return nil, err
	} else {
		s := &systemconfigupdate.SystemConfigUpdate{
			State: m.gcShow,
		}
		return s, nil
	}
}
func (m *Dummymanager) GcActionStrategy(c *gophercloud.ServiceClient, opts systemconfigupdate.StrategyActionOpts) (*systemconfigupdate.SystemConfigUpdate, error) {
	m.strategyActionSend = true
	if m.strategyActionError {
		err := errors.New("test: action sent error")
		return nil, err
	} else {
		s := &systemconfigupdate.SystemConfigUpdate{}
		return s, nil
	}
}
func (m *Dummymanager) GcCreate(c *gophercloud.ServiceClient, opts systemconfigupdate.SystemConfigUpdateOpts) (*systemconfigupdate.SystemConfigUpdate, error) {
	if m.strategyCreateError {
		return nil, errors.New("Test strategy create error")
	} else {
		m.strategyCreated = true
		m.strategyCreateRequest = opts
		s := &systemconfigupdate.SystemConfigUpdate{}
		return s, nil
	}
}
func (m *Dummymanager) GcDelete(c *gophercloud.ServiceClient) (r systemconfigupdate.DeleteResult) {
	m.strategyDeleted = true
	re := systemconfigupdate.DeleteResult{}
	return re
}
