import { Separator } from "./ui/separator";

interface LineItem {
  name: string;
  description?: string;
  quantity: number;
  price: number;
}

interface OrderSummaryProps {
  merchantName: string;
  totalAmount: number;
  currency?: string;
  lineItems: LineItem[];
}

export function OrderSummary({
  merchantName,
  totalAmount,
  currency = "USD",
  lineItems,
}: OrderSummaryProps) {
  const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: currency,
    }).format(amount);
  };

  return (
    <div className="flex flex-col h-full">
      {/* Merchant Info */}
      <div className="px-6 py-8 lg:px-8 lg:py-10">
        <div className="flex items-center gap-3 mb-8">
          <div className="w-12 h-12 rounded-lg bg-primary flex items-center justify-center">
            <span className="text-primary-foreground">
              {merchantName.charAt(0)}
            </span>
          </div>
          <div>
            <h2 className="text-foreground">{merchantName}</h2>
            <p className="text-sm text-muted-foreground">Secure checkout</p>
          </div>
        </div>

        {/* Total Amount - Highly Prominent */}
        <div className="mb-6">
          <p className="text-sm text-muted-foreground mb-1">Total due</p>
          <div className="text-4xl text-foreground tracking-tight">
            {formatCurrency(totalAmount)}
          </div>
        </div>
      </div>

      <Separator className="bg-border" />

      {/* Line Items */}
      <div className="px-6 py-6 lg:px-8 flex-1 overflow-y-auto">
        <h3 className="text-sm text-muted-foreground mb-4">Order details</h3>
        <div className="space-y-4">
          {lineItems.map((item, index) => (
            <div key={index} className="flex justify-between items-start gap-4">
              <div className="flex-1 min-w-0">
                <p className="text-sm text-foreground">{item.name}</p>
                {item.description && (
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {item.description}
                  </p>
                )}
                {item.quantity > 1 && (
                  <p className="text-xs text-muted-foreground mt-0.5">
                    Qty: {item.quantity}
                  </p>
                )}
              </div>
              <p className="text-sm text-foreground whitespace-nowrap">
                {formatCurrency(item.price * item.quantity)}
              </p>
            </div>
          ))}
        </div>

        <Separator className="my-4 bg-border" />

        {/* Subtotal and Total */}
        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span className="text-muted-foreground">Subtotal</span>
            <span className="text-foreground">
              {formatCurrency(
                lineItems.reduce((acc, item) => acc + item.price * item.quantity, 0)
              )}
            </span>
          </div>
          <div className="flex justify-between">
            <span className="text-foreground">Total</span>
            <span className="text-foreground">{formatCurrency(totalAmount)}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
