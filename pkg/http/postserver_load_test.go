package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
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
	"github.com/gin-gonic/gin"
	gm "github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTP", func() {
	var (
		mockCtrl       *gm.Controller // gomock struct
		mockLog        *mock_logr.MockLogger
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
		mockLog = mock_logr.NewMockLogger(mockCtrl)
		log = mockLog
		mockController = mock_lifecycle.NewMockController(mockCtrl)
		executionID = uuid.New().String()
		node = uuid.New().String()
		cfg = &config.Config{
			Metrics: config.Metrics{
				Prefix: "foo",
			},
		}

		s = &PostServer{
			ReportPath: tempDir(executionID),
		}
		s.InjectController(mockController)
		s.InjectConfig(cfg)

		rr = httptest.NewRecorder()

		// Need to create a router that we can pass the request through so that the vars will be added to the context
		router = gin.New()
		path = fmt.Sprintf("/report/%s/%s%s", node, executionID, CallbackBaseResultSubPath)
	})
	AfterEach(func() {
		_ = os.RemoveAll(s.ReportPath)
	})
	// disable by default
	XIt("generate parallel load", func() {
		path = fmt.Sprintf("/report/%s/%s%s", node, executionID, CallbackBaseFileSubPath)
		router.POST(CallbackBasePath+CallbackBaseFileSubPath, s.postFile)

		file := filepath.Join(s.ReportPath, "file.txt")
		fileSizeMB := 50
		sleep := 2 * time.Millisecond
		loops := 200

		cmd := exec.Command("dd", "if=/dev/urandom", fmt.Sprintf("of=%s", file), "bs=1M", fmt.Sprintf("count=%d", fileSizeMB)) // #nosec G204:
		_, err := cmd.Output()
		Ω(err).ShouldNot(HaveOccurred())

		data, err := ioutil.ReadFile(file)
		Ω(err).ShouldNot(HaveOccurred())

		mockLog.EXPECT().WithValues("node", node, "id", executionID).Return(mockLog).Times(loops)
		mockLog.EXPECT().WithValues("name", gm.Any(), "path", gm.Any(), "length", gm.Any()).Return(mockLog).Times(loops)

		mockController.EXPECT().ReportReceived(executionID, node, gm.Any(), gm.Any()).Times(loops)
		mockLog.EXPECT().WithValues("name", gm.Any(), "path", gm.Any()).Return(mockLog).Times(loops)
		mockLog.EXPECT().Info("received file").Times(loops)

		var wg sync.WaitGroup
		for i := 0; i < loops; i++ {
			wg.Add(1)
			time.Sleep(sleep)
			ii := i
			go func() {
				req, err := http.NewRequest("POST", path, bytes.NewBuffer(data))
				Ω(err).ShouldNot(HaveOccurred())

				req.Header.Add("content-type", "application/json")
				req.Header.Add("Content-Disposition", fmt.Sprintf(`attachment;filename="%d.txt"`, ii))
				router.ServeHTTP(rr, req)
				defer wg.Done()
			}()
		}
		wg.Wait()
	})
})
