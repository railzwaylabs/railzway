import { useState } from "react";
import { OrderSummary } from "./OrderSummary";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { CreditCard, Lock } from "lucide-react";

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

export function CheckoutVariantB() {
  const [cardNumber, setCardNumber] = useState("");
  const [cardExpiry, setCardExpiry] = useState("");
  const [cardCvc, setCardCvc] = useState("");
  const [cardName, setCardName] = useState("");

  const formatCardNumber = (value: string) => {
    const v = value.replace(/\s+/g, "").replace(/[^0-9]/gi, "");
    const matches = v.match(/\d{4,16}/g);
    const match = (matches && matches[0]) || "";
    const parts = [];

    for (let i = 0, len = match.length; i < len; i += 4) {
      parts.push(match.substring(i, i + 4));
    }

    if (parts.length) {
      return parts.join(" ");
    } else {
      return value;
    }
  };

  const formatExpiry = (value: string) => {
    const v = value.replace(/\s+/g, "").replace(/[^0-9]/gi, "");
    if (v.length >= 2) {
      return v.substring(0, 2) + " / " + v.substring(2, 4);
    }
    return v;
  };

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

          {/* Right Column - Direct Payment Form */}
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

              <h1 className="mb-2">Payment details</h1>
              <p className="text-sm text-muted-foreground mb-8">
                Enter your card information to complete your purchase
              </p>

              <form className="space-y-6">
                {/* Card Information Section */}
                <div className="space-y-4">
                  <div>
                    <Label htmlFor="cardNumber">Card number</Label>
                    <div className="relative mt-2">
                      <Input
                        id="cardNumber"
                        type="text"
                        placeholder="1234 1234 1234 1234"
                        value={cardNumber}
                        onChange={(e) =>
                          setCardNumber(formatCardNumber(e.target.value))
                        }
                        maxLength={19}
                        className="h-12 pr-12 bg-input-background border-border focus-visible:ring-2 focus-visible:ring-ring"
                      />
                      <div className="absolute right-3 top-1/2 -translate-y-1/2">
                        <CreditCard className="w-5 h-5 text-muted-foreground" />
                      </div>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <Label htmlFor="cardExpiry">Expiration</Label>
                      <Input
                        id="cardExpiry"
                        type="text"
                        placeholder="MM / YY"
                        value={cardExpiry}
                        onChange={(e) =>
                          setCardExpiry(formatExpiry(e.target.value))
                        }
                        maxLength={7}
                        className="h-12 mt-2 bg-input-background border-border focus-visible:ring-2 focus-visible:ring-ring"
                      />
                    </div>
                    <div>
                      <Label htmlFor="cardCvc">CVC</Label>
                      <Input
                        id="cardCvc"
                        type="text"
                        placeholder="123"
                        value={cardCvc}
                        onChange={(e) =>
                          setCardCvc(e.target.value.replace(/[^0-9]/gi, ""))
                        }
                        maxLength={4}
                        className="h-12 mt-2 bg-input-background border-border focus-visible:ring-2 focus-visible:ring-ring"
                      />
                    </div>
                  </div>

                  <div>
                    <Label htmlFor="cardName">Cardholder name</Label>
                    <Input
                      id="cardName"
                      type="text"
                      placeholder="Full name on card"
                      value={cardName}
                      onChange={(e) => setCardName(e.target.value)}
                      className="h-12 mt-2 bg-input-background border-border focus-visible:ring-2 focus-visible:ring-ring"
                    />
                  </div>
                </div>

                {/* Pay Button */}
                <Button className="w-full h-12" size="lg" type="submit">
                  Pay $1,499.00
                </Button>

                {/* Security Notice */}
                <div className="flex items-center justify-center gap-2 text-sm text-muted-foreground pt-2">
                  <Lock className="w-4 h-4" />
                  <span>Guaranteed safe & secure checkout</span>
                </div>

                {/* Payment Provider Badge */}
                <div className="pt-6 border-t border-border">
                  <div className="flex items-center justify-center gap-3">
                    <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-muted/50 border border-border">
                      <svg
                        className="w-4 h-4"
                        viewBox="0 0 24 24"
                        fill="none"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path
                          d="M3 10h18M3 14h18"
                          stroke="currentColor"
                          strokeWidth="2"
                          strokeLinecap="round"
                        />
                      </svg>
                      <span className="text-xs">Powered by Stripe</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <svg
                        className="w-8 h-5"
                        viewBox="0 0 48 30"
                        fill="none"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <rect
                          x="1"
                          y="1"
                          width="46"
                          height="28"
                          rx="3"
                          fill="var(--brand-visa)"
                          stroke="var(--brand-card-border)"
                        />
                        <circle cx="18" cy="15" r="7" fill="var(--brand-mastercard-red)" />
                        <circle cx="30" cy="15" r="7" fill="var(--brand-mastercard-orange)" />
                      </svg>
                      <svg
                        className="w-8 h-5"
                        viewBox="0 0 48 30"
                        fill="none"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <rect
                          x="1"
                          y="1"
                          width="46"
                          height="28"
                          rx="3"
                          fill="var(--brand-jcb-blue)"
                          stroke="var(--brand-card-border)"
                        />
                        <path
                          d="M24 8l2.5 7.5h8l-6.5 5 2.5 7.5L24 23l-6.5 5 2.5-7.5-6.5-5h8L24 8z"
                          fill="var(--brand-jcb-gold)"
                        />
                      </svg>
                    </div>
                  </div>
                  <p className="text-xs text-center text-muted-foreground mt-3">
                    Your payment information is encrypted and secure
                  </p>
                </div>
              </form>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
