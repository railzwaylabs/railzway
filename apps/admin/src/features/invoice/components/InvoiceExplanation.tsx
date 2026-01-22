import { useState } from 'react';
import { ChevronDown, ChevronRight, Info } from 'lucide-react';

interface InvoiceExplanationProps {
  invoiceId: string;
}

interface DetailedExplanation {
  quantity: number;
  unit_price: number;
  period_start: string;
  period_end: string;
  source_events: number;
  meter_id?: string;
  meter_name?: string;
  feature_code?: string;
  rating_result_id?: string;
}

interface LineItemExplanation {
  line_item_id: string;
  description: string;
  amount: number;
  explanation?: DetailedExplanation;
}

interface InvoiceExplanationData {
  invoice_id: string;
  total: number;
  breakdown: LineItemExplanation[];
}

export function InvoiceExplanation({ invoiceId }: InvoiceExplanationProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [data, setData] = useState<InvoiceExplanationData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expandedItems, setExpandedItems] = useState<Set<string>>(new Set());

  const fetchExplanation = async () => {
    if (data) {
      setIsOpen(!isOpen);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`/admin/invoices/${invoiceId}/explanation`, {
        credentials: 'include',
      });

      if (!response.ok) {
        throw new Error('Failed to fetch invoice explanation');
      }

      const result = await response.json();
      setData(result);
      setIsOpen(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  const toggleItem = (itemId: string) => {
    const newExpanded = new Set(expandedItems);
    if (newExpanded.has(itemId)) {
      newExpanded.delete(itemId);
    } else {
      newExpanded.add(itemId);
    }
    setExpandedItems(newExpanded);
  };

  const formatCurrency = (amount: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
    }).format(amount / 100);
  };

  const formatDate = (dateStr: string) => {
    if (!dateStr) return 'N/A';
    return new Date(dateStr).toLocaleString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  return (
    <div className="border rounded-lg bg-white">
      <button
        onClick={fetchExplanation}
        disabled={loading}
        className="w-full px-4 py-3 flex items-center justify-between hover:bg-gray-50 transition-colors"
      >
        <div className="flex items-center gap-2">
          <Info className="w-5 h-5 text-blue-600" />
          <span className="font-medium text-gray-900">Invoice Explanation</span>
        </div>
        <div className="flex items-center gap-2">
          {loading && <span className="text-sm text-gray-500">Loading...</span>}
          {isOpen ? (
            <ChevronDown className="w-5 h-5 text-gray-400" />
          ) : (
            <ChevronRight className="w-5 h-5 text-gray-400" />
          )}
        </div>
      </button>

      {error && (
        <div className="px-4 py-3 bg-red-50 border-t border-red-200">
          <p className="text-sm text-red-600">{error}</p>
        </div>
      )}

      {isOpen && data && (
        <div className="border-t">
          <div className="px-4 py-3 bg-gray-50 border-b">
            <div className="flex justify-between items-center">
              <span className="text-sm font-medium text-gray-700">Total Amount</span>
              <span className="text-lg font-bold text-gray-900">
                {formatCurrency(data.total)}
              </span>
            </div>
          </div>

          <div className="divide-y">
            {data.breakdown.map((item) => (
              <div key={item.line_item_id} className="bg-white">
                <div
                  className="px-4 py-3 flex items-center justify-between cursor-pointer hover:bg-gray-50"
                  onClick={() => item.explanation && toggleItem(item.line_item_id)}
                >
                  <div className="flex items-center gap-2 flex-1">
                    {item.explanation && (
                      expandedItems.has(item.line_item_id) ? (
                        <ChevronDown className="w-4 h-4 text-gray-400" />
                      ) : (
                        <ChevronRight className="w-4 h-4 text-gray-400" />
                      )
                    )}
                    <div>
                      <p className="text-sm font-medium text-gray-900">{item.description}</p>
                      {item.explanation && (
                        <p className="text-xs text-gray-500 mt-0.5">
                          {item.explanation.quantity.toLocaleString()} × {formatCurrency(item.explanation.unit_price)}
                        </p>
                      )}
                    </div>
                  </div>
                  <span className="text-sm font-semibold text-gray-900">
                    {formatCurrency(item.amount)}
                  </span>
                </div>

                {item.explanation && expandedItems.has(item.line_item_id) && (
                  <div className="px-4 py-3 bg-gray-50 border-t space-y-2">
                    <div className="grid grid-cols-2 gap-4 text-sm">
                      <div>
                        <span className="text-gray-500">Quantity:</span>
                        <span className="ml-2 font-medium text-gray-900">
                          {item.explanation.quantity.toLocaleString()}
                        </span>
                      </div>
                      <div>
                        <span className="text-gray-500">Unit Price:</span>
                        <span className="ml-2 font-medium text-gray-900">
                          {formatCurrency(item.explanation.unit_price)}
                        </span>
                      </div>
                      <div>
                        <span className="text-gray-500">Period Start:</span>
                        <span className="ml-2 font-medium text-gray-900">
                          {formatDate(item.explanation.period_start)}
                        </span>
                      </div>
                      <div>
                        <span className="text-gray-500">Period End:</span>
                        <span className="ml-2 font-medium text-gray-900">
                          {formatDate(item.explanation.period_end)}
                        </span>
                      </div>
                      {item.explanation.meter_name && (
                        <div>
                          <span className="text-gray-500">Meter:</span>
                          <span className="ml-2 font-medium text-gray-900">
                            {item.explanation.meter_name}
                          </span>
                        </div>
                      )}
                      {item.explanation.feature_code && (
                        <div>
                          <span className="text-gray-500">Feature:</span>
                          <span className="ml-2 font-medium text-gray-900">
                            {item.explanation.feature_code}
                          </span>
                        </div>
                      )}
                      <div>
                        <span className="text-gray-500">Source Events:</span>
                        <span className="ml-2 font-medium text-gray-900">
                          {item.explanation.source_events.toLocaleString()}
                        </span>
                      </div>
                      {item.explanation.rating_result_id && (
                        <div className="col-span-2">
                          <span className="text-gray-500">Rating Result ID:</span>
                          <span className="ml-2 font-mono text-xs text-gray-600">
                            {item.explanation.rating_result_id}
                          </span>
                        </div>
                      )}
                    </div>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
