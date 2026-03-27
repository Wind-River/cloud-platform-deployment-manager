/* SPDX-License-Identifier: Apache-2.0 */
/* Copyright(c) 2023-2026 Wind River Systems, Inc. */

package manager

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Monitor", func() {
	Describe("Check return for monitorStrategyState", func() {
		Context("when failing to obtain vim client", func() {
			It("should return false", func() {
				dm := &Dummymanager{vimClientAvailable: false, gcShow: ""}
				got := monitorStrategyState(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyDeleted).To(BeFalse())
			})
		})
		Context("when failing to obtain strategy status", func() {
			It("should return false", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: ""}
				got := monitorStrategyState(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyDeleted).To(BeFalse())
			})
		})
		Context("when status is strategy ready to apply", func() {
			It("should return false", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyReadyToApply}
				got := monitorStrategyState(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyActionSend).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeFalse())
			})
		})
		Context("when strategy apply error occurs before retry exceeds", func() {
			It("should return false and strategy action not sent", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyReadyToApply, strategyActionError: true, retryCount: 10}
				got := monitorStrategyState(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyActionSend).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeFalse())
			})
		})
		Context("when strategy apply error occurs and retry exceeds", func() {
			It("should return true, strategy action not sent and strategy deleted", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyReadyToApply, strategyActionError: true, retryCount: DefaultMaxStrategyRetryCount + 1}
				got := monitorStrategyState(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyActionSend).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeTrue())
			})
		})
		Context("when status is build failed", func() {
			It("should return true and strategy deleted", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyBuildFailed}
				got := monitorStrategyState(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeTrue())
			})
		})
		Context("when status is strategy apply failed", func() {
			It("should return true and strategy deleted", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyApplyFailed}
				got := monitorStrategyState(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeTrue())
			})
		})
		Context("when status is strategy applying", func() {
			It("should return false", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyApplying}
				got := monitorStrategyState(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyDeleted).To(BeFalse())
			})
		})
		Context("when status is strategy build timeout", func() {
			It("should return true and strategy deleted", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyBuildTimeout}
				got := monitorStrategyState(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeTrue())
			})
		})
		Context("when status is strategy apply timeout", func() {
			It("should return true and strategy deleted", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyApplyTimeout}
				got := monitorStrategyState(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeTrue())
			})
		})
		Context("when status is strategy abort failed", func() {
			It("should return true and strategy deleted", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyAbortFailed}
				got := monitorStrategyState(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeTrue())
			})
		})
		Context("when status is strategy aborting", func() {
			It("should return false", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyAborting}
				got := monitorStrategyState(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyDeleted).To(BeFalse())
			})
		})
		Context("when status is strategy abort timeout", func() {
			It("should return true and strategy deleted", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyAbortTimeout}
				got := monitorStrategyState(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeTrue())
			})
		})
		Context("when status is strategy applied", func() {
			It("should return true and strategy deleted", func() {
				dm := &Dummymanager{vimClientAvailable: true, gcShow: StrategyApplied}
				got := monitorStrategyState(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyDeleted).To(BeTrue())
			})
		})
	})

	Describe("Check return for ManageStrategy", func() {
		Context("when monitor version does not match config version", func() {
			It("should return false", func() {
				dm := &Dummymanager{config_version: 0, monitor_version: 1}
				got := ManageStrategy(dm)
				Expect(got).To(BeFalse())
			})
		})
		Context("when reconcile is not finished", func() {
			It("should return false and strategy not created", func() {
				rsc := map[string]*ResourceInfo{
					"controller-0": {
						StrategyRequired: StrategyNotRequired,
						Reconciled:       false,
					},
				}
				dm := &Dummymanager{strategySent: false, Resource: rsc}
				got := ManageStrategy(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyCreated).To(BeFalse())
			})
		})
		Context("when reconcile is finished but no strategy required", func() {
			It("should return false and strategy not created", func() {
				rsc := map[string]*ResourceInfo{
					"controller-0": {
						StrategyRequired: StrategyNotRequired,
						Reconciled:       true,
					},
				}
				dm := &Dummymanager{strategySent: false, Resource: rsc}
				got := ManageStrategy(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyCreated).To(BeFalse())
			})
		})
		Context("when lock is required for system", func() {
			It("should return false and strategy created and sent", func() {
				rsc := map[string]*ResourceInfo{
					"controller-0": {
						ResourceType:     ResourceSystem,
						StrategyRequired: StrategyLockRequired,
						Reconciled:       true,
					},
				}
				dm := &Dummymanager{strategySent: false, Resource: rsc, vimClientAvailable: true}
				got := ManageStrategy(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyCreated).To(BeTrue())
				Expect(dm.strategySent).To(BeTrue())
				Expect(dm.strategyCreateRequest.ControllerApplyType).To(Equal("serial"))
				Expect(dm.strategyCreateRequest.WorkerApplyType).To(Equal("parallel"))
				Expect(dm.strategyCreateRequest.StorageApplyType).To(Equal("ignore"))
			})
		})
		Context("when lock is required for controller", func() {
			It("should return false and strategy created and sent", func() {
				rsc := map[string]*ResourceInfo{
					"controller-0": {
						ResourceType:     ResourceHost,
						Personality:      PersonalityController,
						StrategyRequired: StrategyLockRequired,
						Reconciled:       true,
					},
				}
				dm := &Dummymanager{strategySent: false, Resource: rsc, vimClientAvailable: true}
				got := ManageStrategy(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyCreated).To(BeTrue())
				Expect(dm.strategySent).To(BeTrue())
				Expect(dm.strategyCreateRequest.ControllerApplyType).To(Equal("serial"))
				Expect(dm.strategyCreateRequest.WorkerApplyType).To(Equal("parallel"))
				Expect(dm.strategyCreateRequest.StorageApplyType).To(Equal("ignore"))
			})
		})
		Context("when lock is required for worker", func() {
			It("should return false and strategy created and sent", func() {
				rsc := map[string]*ResourceInfo{
					"controller-0": {
						ResourceType:     ResourceHost,
						Personality:      PersonalityWorker,
						StrategyRequired: StrategyLockRequired,
						Reconciled:       true,
					},
				}
				dm := &Dummymanager{strategySent: false, Resource: rsc, vimClientAvailable: true}
				got := ManageStrategy(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyCreated).To(BeTrue())
				Expect(dm.strategySent).To(BeTrue())
				Expect(dm.strategyCreateRequest.ControllerApplyType).To(Equal("ignore"))
				Expect(dm.strategyCreateRequest.WorkerApplyType).To(Equal("parallel"))
				Expect(dm.strategyCreateRequest.StorageApplyType).To(Equal("ignore"))
			})
		})
		Context("when lock is required for storage", func() {
			It("should return false and strategy created and sent", func() {
				rsc := map[string]*ResourceInfo{
					"controller-0": {
						ResourceType:     ResourceHost,
						Personality:      PersonalityStorage,
						StrategyRequired: StrategyLockRequired,
						Reconciled:       true,
					},
				}
				dm := &Dummymanager{strategySent: false, Resource: rsc, vimClientAvailable: true}
				got := ManageStrategy(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyCreated).To(BeTrue())
				Expect(dm.strategySent).To(BeTrue())
				Expect(dm.strategyCreateRequest.ControllerApplyType).To(Equal("ignore"))
				Expect(dm.strategyCreateRequest.WorkerApplyType).To(Equal("ignore"))
				Expect(dm.strategyCreateRequest.StorageApplyType).To(Equal("serial"))
			})
		})
		Context("when strategy create error occurs before retry exceeds", func() {
			It("should return false and strategy not created", func() {
				rsc := map[string]*ResourceInfo{
					"controller-0": {
						ResourceType:     ResourceHost,
						Personality:      PersonalityController,
						StrategyRequired: StrategyLockRequired,
						Reconciled:       true,
					},
				}
				dm := &Dummymanager{strategySent: false, Resource: rsc, vimClientAvailable: true, strategyCreateError: true, retryCount: 10}
				got := ManageStrategy(dm)
				Expect(got).To(BeFalse())
				Expect(dm.strategyCreated).To(BeFalse())
			})
		})
		Context("when strategy create error occurs and retry exceeds", func() {
			It("should return false and strategy not created", func() {
				rsc := map[string]*ResourceInfo{
					"controller-0": {
						ResourceType:     ResourceHost,
						Personality:      PersonalityController,
						StrategyRequired: StrategyLockRequired,
						Reconciled:       true,
					},
				}
				dm := &Dummymanager{strategySent: false, Resource: rsc, vimClientAvailable: true, strategyCreateError: true, retryCount: DefaultMaxStrategyRetryCount + 1}
				got := ManageStrategy(dm)
				Expect(got).To(BeTrue())
				Expect(dm.strategyCreated).To(BeFalse())
			})
		})
		Context("when strategy is applying after strategy sent", func() {
			It("should return false", func() {
				dm := &Dummymanager{strategySent: true, vimClientAvailable: true, gcShow: StrategyApplying}
				got := ManageStrategy(dm)
				Expect(got).To(BeFalse())
			})
		})
		Context("when strategy is applied after strategy sent", func() {
			It("should return true", func() {
				dm := &Dummymanager{strategySent: true, vimClientAvailable: true, gcShow: StrategyApplied}
				got := ManageStrategy(dm)
				Expect(got).To(BeTrue())
			})
		})
	})
})
