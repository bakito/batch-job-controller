package lifecycle

import (
	"math/rand"
	"os"
	"path/filepath"
	"runtime"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/bakito/batch-job-controller/pkg/metrics"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("lifecycle", func() {
	var (
		cfg       *config.Config
		pc        *metrics.Collector
		repDir    string
		namespace string
		poolSize  int
	)
	BeforeEach(func() {
		repDir = "test-" + uuid.New().String()
		namespace = uuid.New().String()
		poolSize = rand.Int() // #nosec G404 ok for tests
		cfg = &config.Config{
			Namespace:       namespace,
			ReportDirectory: repDir,
			PodPoolSize:     poolSize,
			Metrics: config.Metrics{
				Prefix: "foo",
			},
		}
		pc, _ = metrics.NewPromCollector(cfg)
	})
	Context("NewController", func() {
		It("should return a new controller", func() {
			c := NewController(cfg, pc)
			Ω(c).ShouldNot(BeNil())
			Ω(c.(*controller).config).ShouldNot(BeNil())
			Ω(c.(*controller).log).ShouldNot(BeNil())
			Ω(c.(*controller).reportDir).Should(Equal(repDir))
			Ω(c.(*controller).podPoolSize).Should(Equal(poolSize))
		})
	})
	Context("NewExecution", func() {
		var c *controller
		BeforeEach(func() {
			cfg.PodPoolSize = 0
			c = NewController(cfg, pc).(*controller)
		})
		AfterEach(func() {
			os.RemoveAll(c.reportDir)
		})
		It("should create an id and directory", func() {
			id := c.NewExecution(0)
			Ω(id).ShouldNot(BeEmpty())
			_, err := os.Stat(filepath.Join(repDir, id))
			Ω(err).ShouldNot(HaveOccurred())
			if runtime.GOOS != "windows" {
				_, err = os.Lstat(filepath.Join(repDir, "latest"))
				Ω(err).ShouldNot(HaveOccurred())
			}
		})
		It("should create an id and directory and move the link", func() {
			id1 := c.NewExecution(0)
			Ω(id1).ShouldNot(BeEmpty())
			_, err := os.Stat(filepath.Join(repDir, id1))
			Ω(err).ShouldNot(HaveOccurred())
			if runtime.GOOS != "windows" {
				_, err = os.Lstat(filepath.Join(repDir, "latest"))
				Ω(err).ShouldNot(HaveOccurred())
			}
			id2 := c.NewExecution(0)
			Ω(id2).ShouldNot(BeEmpty())
			_, err = os.Lstat(filepath.Join(repDir, id2))
			Ω(err).ShouldNot(HaveOccurred())
		})
	})
})
