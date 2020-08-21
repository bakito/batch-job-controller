package http

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	mock_cache "github.com/bakito/batch-job-controller/pkg/mocks/cache"
	mock_logr "github.com/bakito/batch-job-controller/pkg/mocks/logr"
	gm "github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	reportJSON = `{ "test": [{ "value": 1.0, "labels": { "label_a": "AAA", "label_b": "BBB" }}] }`
)

var _ = Describe("HTTP", func() {
	var (
		mockCtrl    *gm.Controller //gomock struct
		mockLog     *mock_logr.MockLogger
		mockCache   *mock_cache.MockCache
		executionID string
		node        string

		s *PostServer

		rr     *httptest.ResponseRecorder
		router *mux.Router
	)
	BeforeEach(func() {
		mockCtrl = gm.NewController(GinkgoT())
		mockLog = mock_logr.NewMockLogger(mockCtrl)
		log = mockLog
		mockCache = mock_cache.NewMockCache(mockCtrl)
		executionID = uuid.New().String()
		node = uuid.New().String()
		s = &PostServer{
			ReportPath: tempDir(executionID),
			Cache:      mockCache,
		}

		rr = httptest.NewRecorder()

		// Need to create a router that we can pass the request through so that the vars will be added to the context
		router = mux.NewRouter()

	})
	AfterEach(func() {
		os.RemoveAll(s.ReportPath)
	})
	Context("postReport", func() {
		var (
			path string
		)
		BeforeEach(func() {
			path = fmt.Sprintf("/report/%s/%s%s", node, executionID, CallbackBaseResultSubPath)
			router.HandleFunc(CallbackBasePath+CallbackBaseResultSubPath, s.postReport)

			mockLog.EXPECT().WithValues("node", node, "id", executionID, "length", gm.Any()).Return(mockLog)
		})
		It("succeed if file is saved", func() {

			mockCache.EXPECT().ReportReceived(executionID, node, gm.Any(), gm.Any())
			mockLog.EXPECT().WithValues("name", gm.Any(), "path", gm.Any()).Return(mockLog)
			mockLog.EXPECT().Info("received report")

			req, err := http.NewRequest("POST", path, strings.NewReader(reportJSON))
			Ω(err).ShouldNot(HaveOccurred())

			router.ServeHTTP(rr, req)

			Ω(rr.Code).Should(Equal(http.StatusOK))

			files, err := ioutil.ReadDir(filepath.Join(s.ReportPath, executionID))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(files).Should(HaveLen(1))

			b, err := ioutil.ReadFile(filepath.Join(s.ReportPath, executionID, files[0].Name()))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(b).Should(Equal([]byte(reportJSON)))
		})
		It("fails if json is invalid", func() {

			mockCache.EXPECT().ReportReceived(executionID, node, gm.Any(), gm.Any())
			mockLog.EXPECT().WithValues("result", gm.Any()).Return(mockLog)
			mockLog.EXPECT().Error(gm.Any(), gm.Any())

			req, err := http.NewRequest("POST", path, strings.NewReader("foo"))
			Ω(err).ShouldNot(HaveOccurred())

			router.ServeHTTP(rr, req)

			Ω(rr.Code).Should(Equal(http.StatusBadRequest))

			files, err := ioutil.ReadDir(filepath.Join(s.ReportPath, executionID))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(files).Should(HaveLen(0))
		})
	})

	Context("postFile", func() {
		var (
			path                   string
			fileName               string
			generatedFileExtension string
		)
		BeforeEach(func() {
			fileName = uuid.New().String() + ".txt"
			path = fmt.Sprintf("/report/%s/%s%s", node, executionID, CallbackBaseFileSubPath)
			router.HandleFunc(CallbackBasePath+CallbackBaseFileSubPath, s.postFile)

			mockLog.EXPECT().WithValues("node", node, "id", executionID, "name", gm.Any(), "path", gm.Any(), "length", gm.Any()).Return(mockLog)

			mockCache.EXPECT().ReportReceived(executionID, node, gm.Any(), gm.Any())
			mockLog.EXPECT().WithValues("name", gm.Any(), "path", gm.Any()).Return(mockLog)
			mockLog.EXPECT().Info("received file")
		})
		AfterEach(func() {
			Ω(rr.Code).Should(Equal(http.StatusOK))

			files, err := ioutil.ReadDir(filepath.Join(s.ReportPath, executionID))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(files).Should(HaveLen(1))
			if generatedFileExtension != "" {
				Ω(files[0].Name()).Should(HavePrefix(node + "-"))
				Ω(files[0].Name()).Should(HaveSuffix(generatedFileExtension))
			} else {
				Ω(files[0].Name()).Should(Equal(node + "-" + fileName))
			}

			b, err := ioutil.ReadFile(filepath.Join(s.ReportPath, executionID, files[0].Name()))
			Ω(err).ShouldNot(HaveOccurred())
			Ω(b).Should(Equal([]byte("foo")))
		})
		It("succeed if file is saved with correct name from query parameter", func() {
			req, err := http.NewRequest("POST", fmt.Sprintf("%s?name=%s", path, fileName), strings.NewReader("foo"))
			Ω(err).ShouldNot(HaveOccurred())
			router.ServeHTTP(rr, req)
		})
		It("succeed if file is saved with correct name from header", func() {
			req, err := http.NewRequest("POST", path, strings.NewReader("foo"))
			req.Header.Add("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, fileName))
			Ω(err).ShouldNot(HaveOccurred())
			router.ServeHTTP(rr, req)
		})
		It("succeed if file is saved with generated name with .file extension", func() {
			generatedFileExtension = ".file"
			req, err := http.NewRequest("POST", path, strings.NewReader("foo"))
			Ω(err).ShouldNot(HaveOccurred())
			router.ServeHTTP(rr, req)
		})
		It("succeed if file is saved with generated name with .txt extension", func() {
			generatedFileExtension = ".txt"
			req, err := http.NewRequest("POST", path, strings.NewReader("foo"))
			req.Header.Add("Content-Type", "text/plain")
			Ω(err).ShouldNot(HaveOccurred())
			router.ServeHTTP(rr, req)
		})
		It("succeed if file is saved with generated name with .json extension", func() {
			generatedFileExtension = ".json"
			req, err := http.NewRequest("POST", path, strings.NewReader("foo"))
			req.Header.Add("content-type", "application/json")
			Ω(err).ShouldNot(HaveOccurred())
			router.ServeHTTP(rr, req)
		})

	})
})

func tempDir(id string) string {
	dir, err := ioutil.TempDir("", "go-test-")
	Ω(err).ShouldNot(HaveOccurred())
	err = os.MkdirAll(filepath.Join(dir, id), os.ModePerm)
	Ω(err).ShouldNot(HaveOccurred())
	return dir
}
