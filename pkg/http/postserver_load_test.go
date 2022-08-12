package http

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/bakito/batch-job-controller/pkg/config"
	mock_lifecycle "github.com/bakito/batch-job-controller/pkg/mocks/lifecycle"
	mock_logr "github.com/bakito/batch-job-controller/pkg/mocks/logr"
	"github.com/bakito/batch-job-controller/pkg/test"
	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	gm "github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// disable by default
var _ = XDescribe("HTTP", func() {
	var (
		mockCtrl       *gm.Controller // gomock struct
		mockSink       *mock_logr.MockLogSink
		mockController *mock_lifecycle.MockController
		executionID    string
		node           string

		s   *PostServer
		cfg *config.Config

		rr     *httptest.ResponseRecorder
		router *gin.Engine
		path   string
	)
	BeforeEach(func() {
		gin.SetMode(gin.ReleaseMode)
		mockCtrl = gm.NewController(GinkgoT())
		mockSink = mock_logr.NewMockLogSink(mockCtrl)
		mockController = mock_lifecycle.NewMockController(mockCtrl)
		executionID = uuid.New().String()
		node = uuid.New().String()
		tmp, err := test.TempDir(executionID)
		Ω(err).ShouldNot(HaveOccurred())
		cfg = &config.Config{
			Metrics: config.Metrics{
				Prefix: "foo",
			},
			ReportDirectory: tmp,
		}

		mockSink.EXPECT().Init(gm.Any())
		mockSink.EXPECT().Enabled(gm.Any()).AnyTimes().Return(true)
		s = &PostServer{
			Config: cfg,
			Server: &Server{
				Log:    logr.New(mockSink),
				Config: cfg,
			},
		}
		s.InjectController(mockController)
		s.InjectConfig(cfg)

		rr = httptest.NewRecorder()

		// Need to create a router that we can pass the request through so that the vars will be added to the context
		router = gin.New()
		path = fmt.Sprintf("/report/%s/%s%s", node, executionID, CallbackBaseResultSubPath)
		DeferCleanup(func() error {
			return os.RemoveAll(s.Config.ReportDirectory)
		})
	})
	It("generate parallel load", func() {
		path = fmt.Sprintf("/report/%s/%s%s", node, executionID, CallbackBaseFileSubPath)
		router.POST(CallbackBasePath+CallbackBaseFileSubPath, s.postFile)

		file := filepath.Join(s.Config.ReportDirectory, "file.txt")
		fileSizeMB := 50
		sleep := 2 * time.Millisecond
		loops := 200

		cmd := exec.Command("dd", "if=/dev/urandom", fmt.Sprintf("of=%s", file), "bs=1M", fmt.Sprintf("count=%d", fileSizeMB)) // #nosec G204:
		_, err := cmd.Output()
		Ω(err).ShouldNot(HaveOccurred())

		data, err := os.ReadFile(file)
		Ω(err).ShouldNot(HaveOccurred())

		mockSink.EXPECT().WithValues("node", node, "id", executionID).Return(mockSink).Times(loops)
		mockSink.EXPECT().WithValues("name", gm.Any(), "path", gm.Any(), "length", gm.Any()).Return(mockSink).Times(loops)

		mockSink.EXPECT().Info(gm.Any(), "received 1 file").Times(loops)

		var wg sync.WaitGroup
		for i := 0; i < loops; i++ {
			wg.Add(1)
			time.Sleep(sleep)
			ii := i
			go func() {
				defer wg.Done()
				defer GinkgoRecover()
				req, err := http.NewRequest("POST", path, bytes.NewBuffer(data))
				Ω(err).ShouldNot(HaveOccurred())

				req.Header.Add("content-type", "application/json")
				req.Header.Add("Content-Disposition", fmt.Sprintf(`attachment;filename="%d.txt"`, ii))
				router.ServeHTTP(rr, req)
			}()
		}
		wg.Wait()
	})
})
