import { useState } from "react";
import { OrderSummary } from "./OrderSummary";
import { Button } from "./ui/button";
import { Card } from "./ui/card";
import { RadioGroup, RadioGroupItem } from "./ui/radio-group";
import { Label } from "./ui/label";
import { CreditCard, Building2, Wallet, ChevronRight } from "lucide-react";

const mockLineItems = [
  {
    name: "Pro Plan Subscription",
    description: "Annual billing",
    quantity: 1,
    price: 299.0,
  },
  {
    name: "Additional User Seats",
    description: "5 users @ $20/month",
    quantity: 12,
    price: 100.0,
  },
];

export function CheckoutVariantA() {
  const [selectedMethod, setSelectedMethod] = useState<string>("stripe");

  const paymentMethods = [
    {
      id: "stripe",
      name: "Credit or Debit Card",
      description: "Visa, Mastercard, Amex, Discover",
      icon: CreditCard,
      provider: "Stripe",
    },
    {
      id: "xendit",
      name: "Virtual Account",
      description: "Bank transfer via virtual account",
      icon: Building2,
      provider: "Xendit",
    },
    {
      id: "midtrans",
      name: "E-Wallets & QRIS",
      description: "GoPay, OVO, Dana, ShopeePay",
      icon: Wallet,
      provider: "Midtrans",
    },
  ];

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <div className="w-full max-w-6xl">
        <div className="grid lg:grid-cols-2 gap-0 lg:gap-8">
          {/* Left Column - Order Summary */}
          <div className="hidden lg:block bg-muted/30 rounded-lg border border-border overflow-hidden">
            <OrderSummary
              merchantName="Acme Corporation"
              totalAmount={1499.0}
              lineItems={mockLineItems}
            />
          </div>

          {/* Right Column - Payment Method Selection */}
          <div className="bg-background rounded-lg border border-border p-6 lg:p-8">
            <div className="max-w-md mx-auto">
              {/* Mobile Order Summary - Top Position */}
              <div className="lg:hidden mb-8 pb-8 border-b border-border">
                <div className="bg-muted/30 rounded-lg border border-border -mx-2">
                  <OrderSummary
                    merchantName="Acme Corporation"
                    totalAmount={1499.0}
                    lineItems={mockLineItems}
                  />
                </div>
              </div>

              <h1 className="mb-2">Choose payment method</h1>
              <p className="text-sm text-muted-foreground mb-8">
                Select your preferred payment option to continue
              </p>

              <div className="space-y-3 mb-8">
                {paymentMethods.map((method) => (
                  <Card
                    key={method.id}
                    className={`p-4 cursor-pointer transition-all border-2 hover:bg-muted/50 ${
                      selectedMethod === method.id
                        ? "border-primary bg-muted/30"
                        : "border-border"
                    }`}
                    onClick={() => setSelectedMethod(method.id)}
                  >
                    <div className="flex items-start gap-4">
                      <div className="flex items-center justify-center w-10 h-10 rounded-md bg-muted shrink-0">
                        <method.icon className="w-5 h-5 text-foreground" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <span className="text-foreground">{method.name}</span>
                        </div>
                        <p className="text-sm text-muted-foreground">
                          {method.description}
                        </p>
                      </div>
                      <div
                        className={`flex items-center justify-center w-5 h-5 rounded-full border-2 shrink-0 transition-colors ${
                          selectedMethod === method.id
                            ? "border-primary"
                            : "border-border"
                        }`}
                      >
                        <div
                          className={`w-2.5 h-2.5 rounded-full bg-primary transition-opacity ${
                            selectedMethod === method.id ? "opacity-100" : "opacity-0"
                          }`}
                        />
                      </div>
                    </div>
                  </Card>
                ))}
              </div>

              {/* Continue Button */}
              <Button className="w-full h-12 gap-2" size="lg">
                Continue to Payment
                <ChevronRight className="w-4 h-4" />
              </Button>

              {/* Security Badge */}
              <div className="mt-8 pt-6 border-t border-border">
                <div className="flex items-center justify-center gap-2 text-xs text-muted-foreground">
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    className="w-4 h-4"
                  >
                    <rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
                    <path d="M7 11V7a5 5 0 0 1 10 0v4" />
                  </svg>
                  Secured by industry-standard encryption
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
