package lifecycle

import (
	"math/rand"
	"os"
	"path/filepath"

	"github.com/bakito/batch-job-controller/pkg/config"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("lifecycle", func() {
	var (
		cfg       *config.Config
		pc        *Collector
		repDir    string
		namespace string
		poolSize  int
	)
	BeforeEach(func() {
		repDir = "test-" + uuid.New().String()
		namespace = uuid.New().String()
		poolSize = rand.Int()
		cfg = &config.Config{
			Namespace:       namespace,
			ReportDirectory: repDir,
			PodPoolSize:     poolSize,
			Metrics: config.Metrics{
				Prefix: "foo",
			},
		}
		pc, _ = NewPromCollector(cfg)
	})
	Context("NewCache", func() {
		It("should return a new cache", func() {
			c := NewCache(cfg, pc)
			Ω(c).ShouldNot(BeNil())
			Ω(c.(*cache).config).ShouldNot(BeNil())
			Ω(c.(*cache).log).ShouldNot(BeNil())
			Ω(c.(*cache).reportDir).Should(Equal(repDir))
			Ω(c.(*cache).podPoolSize).Should(Equal(poolSize))
		})
	})
	Context("NewExecution", func() {
		var (
			c *cache
		)
		BeforeEach(func() {
			cfg.PodPoolSize = 0
			c = NewCache(cfg, pc).(*cache)
		})
		AfterEach(func() {
			os.RemoveAll(c.reportDir)
		})
		It("should create an id and directory", func() {
			id := c.NewExecution()
			Ω(id).ShouldNot(BeEmpty())
			_, err := os.Stat(filepath.Join(repDir, id))
			Ω(err).ShouldNot(HaveOccurred())
			_, err = os.Stat(filepath.Join(repDir, "latest"))
			Ω(err).ShouldNot(HaveOccurred())
		})
		It("should create an id and directory and move the link", func() {
					id1 := c.NewExecution()
					Ω(id).ShouldNot(BeEmpty())
					_, err := os.Stat(filepath.Join(repDir, id1))
					Ω(err).ShouldNot(HaveOccurred())
					_, err = os.Stat(filepath.Join(repDir, "latest"))
					Ω(err).ShouldNot(HaveOccurred())
					id2 := c.NewExecution()
					Ω(id).ShouldNot(BeEmpty())
					_, err = os.Stat(filepath.Join(repDir, id2))
					Ω(err).ShouldNot(HaveOccurred())
				})
	})
})
