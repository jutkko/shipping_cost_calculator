package calculator_test

import (
	"errors"

	. "github.com/jutinko/shipping_cost_calculator/calculator"
	"github.com/jutinko/shipping_cost_calculator/calculator/fakes"
	"github.com/jutinko/shipping_cost_calculator/utilities"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OrderCalculator", func() {
	var (
		fakeProductStore       *fakes.FakeProductStore
		fakeShippingCalculator *fakes.FakeShippingCalculator
		fakeCurrencyConverter  *fakes.FakeCurrencyConverter
		orderCalculator        *OrderCalculator
		orders                 []*ProductOrder
	)

	BeforeEach(func() {
		fakeProductStore = new(fakes.FakeProductStore)
		fakeShippingCalculator = new(fakes.FakeShippingCalculator)
		fakeCurrencyConverter = new(fakes.FakeCurrencyConverter)
		orderCalculator = NewOrderCalculator(fakeProductStore, fakeShippingCalculator, fakeCurrencyConverter)
	})

	AfterEach(func() {
		orders = []*ProductOrder{}
	})

	Describe("GetPrice", func() {
		It("deligates the call to product store", func() {
			orders = append(orders, NewProductOrder(20, 2))
			fakeProductStore.GetReturns(&utilities.Product{
				Sku:    20,
				Price:  14.4,
				Weight: 0.4,
				Volume: 0.99,
			}, nil)

			_, err := orderCalculator.GetPrice(orders)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeProductStore.GetCallCount()).To(Equal(1))
			Expect(fakeProductStore.GetArgsForCall(0)).To(Equal(20))
		})

		Context("when the product store returns an error", func() {
			It("returns the error", func() {
				orders = append(orders, NewProductOrder(20, 2))
				fakeProductStore.GetReturns(&utilities.Product{}, errors.New("no-product"))
				_, err := orderCalculator.GetPrice(orders)
				Expect(err).To(MatchError("no-product"))
			})
		})

		It("packages the product to a parcel for shipping calculator", func() {
			orders = append(orders, NewProductOrder(20, 2))
			fakeProductStore.GetReturns(&utilities.Product{
				Sku:    20,
				Price:  14.4,
				Weight: 0.4,
				Volume: 0.99,
			}, nil)

			_, err := orderCalculator.GetPrice(orders)
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeShippingCalculator.CalculateCallCount()).To(Equal(1))
			Expect(fakeShippingCalculator.CalculateArgsForCall(0)).To(Equal(utilities.NewParcel(0.8, 1.98)))
		})

		It("converts the price and the shipping price to the desired currency", func() {
			orders = append(orders, NewProductOrder(20, 2))
			fakeProductStore.GetReturns(&utilities.Product{
				Sku:    20,
				Price:  14.4,
				Weight: 0.4,
				Volume: 0.99,
			}, nil)

			fakeShippingCalculator.CalculateReturns(20)

			_, err := orderCalculator.GetPrice(orders)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeCurrencyConverter.ExchangeCallCount()).To(Equal(1))
			Expect(fakeCurrencyConverter.ExchangeArgsForCall(0)).To(BeNumerically("==", 48.8))
		})

		It("returns the price in the desired currency", func() {
			orders = append(orders, NewProductOrder(20, 2))
			fakeCurrencyConverter.ExchangeReturns(1314)
			fakeProductStore.GetReturns(&utilities.Product{
				Sku:    20,
				Price:  14.4,
				Weight: 0.4,
				Volume: 0.99,
			}, nil)

			price, err := orderCalculator.GetPrice(orders)
			Expect(err).NotTo(HaveOccurred())
			Expect(price).To(BeNumerically("==", 1314))
		})

		Context("when there are multiple orders", func() {
			It("should aggregate the prices", func() {
				orders = append(orders, NewProductOrder(20, 2))
				orders = append(orders, NewProductOrder(14, 3))

				fakeProductStore.GetReturns(&utilities.Product{
					Sku:    20,
					Price:  14.4,
					Weight: 0.4,
					Volume: 0.99,
				}, nil)

				_, err := orderCalculator.GetPrice(orders)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeProductStore.GetCallCount()).To(Equal(2))
				Expect(fakeProductStore.GetArgsForCall(0)).To(Equal(20))
				Expect(fakeProductStore.GetArgsForCall(1)).To(Equal(14))
			})

			Context("when the order list is empty", func() {
				It("should return 0", func() {
					price, err := orderCalculator.GetPrice(orders)
					Expect(err).NotTo(HaveOccurred())
					Expect(fakeProductStore.GetCallCount()).To(Equal(0))
					Expect(price).To(BeNumerically("==", 0))
				})
			})

			It("packages the products to a single parcel for shipping calculator", func() {
				orders = append(orders, NewProductOrder(20, 2))
				orders = append(orders, NewProductOrder(1, 4))

				fakeProductStore.GetStub = func(sku int) (*utilities.Product, error) {
					if sku == 20 {
						return &utilities.Product{Weight: 2, Volume: 4}, nil
					} else if sku == 1 {
						return &utilities.Product{Weight: 9, Volume: 10000}, nil
					}

					return nil, nil
				}

				_, err := orderCalculator.GetPrice(orders)
				Expect(err).NotTo(HaveOccurred())
				Expect(fakeShippingCalculator.CalculateCallCount()).To(Equal(1))
				Expect(fakeShippingCalculator.CalculateArgsForCall(0)).To(Equal(utilities.NewParcel(40, 40008)))
			})

			Context("when the order has more than 15 items", func() {
				It("should use the wholesale price", func() {
					for i := 0; i < 15; i++ {
						orders = append(orders, NewProductOrder(i, 2))
					}
					orders = append(orders, NewProductOrder(1, 19))

					fakeProductStore.GetReturns(&utilities.Product{
						Sku:        20,
						WholePrice: 1,
						Price:      14.4,
						Weight:     0.4,
						Volume:     0.99,
					}, nil)

					fakeShippingCalculator.CalculateReturns(20)

					_, err := orderCalculator.GetPrice(orders)
					Expect(err).NotTo(HaveOccurred())

					Expect(fakeCurrencyConverter.ExchangeCallCount()).To(Equal(1))
					Expect(fakeCurrencyConverter.ExchangeArgsForCall(0)).To(BeNumerically("==", 69))
				})
			})
		})
	})
})