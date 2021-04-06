package cmd

import (
	"context"
	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/lifecycle"
	mock_manager "github.com/bakito/batch-job-controller/pkg/mocks/manager"
	mock_record "github.com/bakito/batch-job-controller/pkg/mocks/record"
	gm "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Main", func() {
	var (
		m                 *Main
		mockCtrl          *gm.Controller //gomock struct
		mockManager       *mock_manager.MockManager
		mockEventRecorder *mock_record.MockEventRecorder
	)

	BeforeEach(func() {
		mockCtrl = gm.NewController(GinkgoT())
		mockManager = mock_manager.NewMockManager(mockCtrl)
		mockEventRecorder = mock_record.NewMockEventRecorder(mockCtrl)
		m = &Main{
			Manager: mockManager,
			Config:  &config.Config{},
		}
		mockManager.EXPECT().GetEventRecorderFor(gm.Any()).Return(mockEventRecorder)
		mockManager.EXPECT().GetAPIReader()
		mockManager.EXPECT().Add(gm.Any())
	})
	Context("addToManager", func() {

		It("should invoke all setter", func() {
			runnable := &r{}
			m.addToManager(runnable)
			Ω(runnable.withConfig).Should(BeTrue())
			Ω(runnable.withController).Should(BeTrue())
			Ω(runnable.withEventRecorder).Should(BeTrue())
			Ω(runnable.withReader).Should(BeTrue())
		})
	})
})

type r struct {
	withConfig        bool
	withController    bool
	withEventRecorder bool
	withReader        bool
}

func (r r) Start(_ context.Context) error {
	return nil
}

// InjectConfig inject the config
func (r *r) InjectConfig(_ *config.Config) {
	r.withConfig = true
}

// InjectController inject the cache
func (r *r) InjectController(_ lifecycle.Controller) {
	r.withController = true
}

// InjectEventRecorder inject the event recorder
func (r *r) InjectEventRecorder(_ record.EventRecorder) {
	r.withEventRecorder = true
}

// InjectReader inject the cache
func (r *r) InjectReader(_ client.Reader) {
	r.withReader = true
}
