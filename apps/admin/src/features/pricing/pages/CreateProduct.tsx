import { useEffect, useState } from "react"
import { useNavigate, useParams } from "react-router-dom"
import { useFieldArray, useForm, useWatch, type Control } from "react-hook-form"

import { admin } from "@/api/client"
import { ForbiddenState } from "@/components/forbidden-state"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Form,
  FormControl,
  FormDescription as FormHint,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Spinner } from "@/components/ui/spinner"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { canManageBilling } from "@/lib/roles"
import { useOrgStore } from "@/stores/orgStore"

type PricingModel = "FLAT" | "USAGE_BASED"
type BillingInterval = "DAY" | "WEEK" | "MONTH" | "YEAR"
type AggregationType = "SUM" | "MAX" | "AVG"

type MetadataEntry = { key: string; value: string }
type FlatAmount = {
  unit_amount: number | null
  effective_from?: string
  effective_to?: string
}
type UsageRate = {
  meter_id: string
  unit_amount: number | null
  minimum_amount: number | null
  maximum_amount: number | null
  aggregation: AggregationType
}
type PriceDraft = {
  name: string
  code?: string
  pricing_model: PricingModel
  billing_interval: BillingInterval
  billing_interval_count: number
  currency: string
  flat: {
    amounts: FlatAmount[]
  }
  usage: {
    rates: UsageRate[]
  }
}

type MeterOption = {
  id: string
  name: string
  code?: string
}

type CreateProductFormValues = {
  product: {
    name: string
    code: string
    description: string
    active: boolean
    metadata: MetadataEntry[]
  }
  prices: PriceDraft[]
}

type StepStatus = "idle" | "loading" | "success" | "error"
type StepState = {
  product: StepStatus
  price: StepStatus
  amount: StepStatus
}

type OrchestrationError =
  | { kind: "product-price"; message: string; detail?: string }
  | { kind: "price-amount"; message: string; detail?: string }
  | { kind: "unknown"; message: string; detail?: string }

const defaultStepState: StepState = {
  product: "idle",
  price: "idle",
  amount: "idle",
}

const pricingModelOptions: Array<{ label: string; value: PricingModel }> = [
  { label: "Flat", value: "FLAT" },
  { label: "Usage-based", value: "USAGE_BASED" },
]

const billingIntervalOptions: Array<{ label: string; value: BillingInterval }> = [
  { label: "Day", value: "DAY" },
  { label: "Week", value: "WEEK" },
  { label: "Month", value: "MONTH" },
  { label: "Year", value: "YEAR" },
]

const aggregationOptions: Array<{ label: string; value: AggregationType }> = [
  { label: "Sum", value: "SUM" },
  { label: "Max", value: "MAX" },
  { label: "Avg", value: "AVG" },
]

const buildPriceCode = (productCode: string, pricingModel: PricingModel) =>
  `${productCode}-${pricingModel.toLowerCase().replace(/_/g, "-")}`

const mapMetadata = (entries: MetadataEntry[]) =>
  entries.reduce<Record<string, string>>((acc, entry) => {
    const key = entry.key.trim()
    if (!key) return acc
    acc[key] = entry.value.trim()
    return acc
  }, {})

const getErrorMessage = (err: unknown, fallback: string) => {
  if (typeof err === "object" && err !== null) {
    const errorMessage = (err as any)?.response?.data?.error?.message
    if (errorMessage) return errorMessage
    const message = (err as any)?.message
    if (message) return message
  }
  return fallback
}

const toNumberOrNull = (value: string) => {
  if (value.trim() === "") return null
  const parsed = Number(value)
  if (Number.isNaN(parsed)) return null
  return parsed
}

const toISODate = (value?: string) => {
  const trimmed = value?.trim()
  if (!trimmed) return undefined
  const parsed = new Date(trimmed)
  if (Number.isNaN(parsed.getTime())) return undefined
  return parsed.toISOString()
}

const formatStepStatus = (status: StepStatus) => {
  switch (status) {
    case "loading":
      return { label: "Saving", variant: "secondary" as const }
    case "success":
      return { label: "Saved", variant: "default" as const }
    case "error":
      return { label: "Failed", variant: "destructive" as const }
    default:
      return { label: "Pending", variant: "outline" as const }
  }
}

const buildDefaultPrice = (): PriceDraft => ({
  name: "",
  code: "",
  pricing_model: "FLAT",
  billing_interval: "MONTH",
  billing_interval_count: 1,
  currency: "USD",
  flat: {
    amounts: [
      {
        unit_amount: null,
        effective_from: "",
        effective_to: "",
      },
    ],
  },
  usage: {
    rates: [
      {
        meter_id: "",
        unit_amount: null,
        minimum_amount: null,
        maximum_amount: null,
        aggregation: "SUM",
      },
    ],
  },
})

export default function CreateProduct() {
  const { orgId } = useParams()
  const navigate = useNavigate()
  const role = useOrgStore((state) => state.currentOrg?.role)
  const canManage = canManageBilling(role)
  const [stepStatus, setStepStatus] = useState<StepState>(defaultStepState)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [orchestrationError, setOrchestrationError] =
    useState<OrchestrationError | null>(null)
  const [createdProductId, setCreatedProductId] = useState<string | null>(null)
  const [createdPriceIds, setCreatedPriceIds] = useState<Record<number, string>>({})
  const [createdAmountIds, setCreatedAmountIds] = useState<Record<number, boolean>>({})
  const [meters, setMeters] = useState<MeterOption[]>([])
  const [metersLoading, setMetersLoading] = useState(false)
  const [metersError, setMetersError] = useState<string | null>(null)

  const form = useForm<CreateProductFormValues>({
    mode: "onBlur",
    defaultValues: {
      product: {
        name: "",
        code: "",
        description: "",
        active: true,
        metadata: [{ key: "", value: "" }],
      },
      prices: [buildDefaultPrice()],
    },
  })

  const productCode = form.watch("product.code")

  const metadataFields = useFieldArray({
    control: form.control,
    name: "product.metadata",
  })
  const priceFields = useFieldArray({
    control: form.control,
    name: "prices",
  })

  useEffect(() => {
    if (!orgId) {
      setMeters([])
      setMetersLoading(false)
      setMetersError(null)
      return
    }

    let isMounted = true
    setMetersLoading(true)
    setMetersError(null)

    admin
      .get("/meters")
      .then((response) => {
        if (!isMounted) return
        setMeters(response.data?.data ?? [])
      })
      .catch((err) => {
        if (!isMounted) return
        setMetersError(err?.message ?? "Unable to load meters.")
      })
      .finally(() => {
        if (!isMounted) return
        setMetersLoading(false)
      })

    return () => {
      isMounted = false
    }
  }, [orgId])

  const setStep = (step: keyof StepState, status: StepStatus) => {
    setStepStatus((prev) => ({ ...prev, [step]: status }))
  }

  const handleNavigateToProduct = (productId: string) => {
    if (!orgId) {
      navigate(`/products/${productId}`)
      return
    }
    navigate(`/orgs/${orgId}/products/${productId}`)
  }

  const createProduct = async (values: CreateProductFormValues) => {
    if (!orgId) {
      throw new Error("Missing organization context.")
    }
    const payload = {
      organization_id: orgId,
      name: values.product.name.trim(),
      code: values.product.code.trim(),
      description: values.product.description.trim() || undefined,
      active: values.product.active,
      metadata: mapMetadata(values.product.metadata),
    }
    const response = await admin.post("/products", payload)
    return response.data?.data
  }

  const createPrice = async (price: PriceDraft, productId: string, code: string) => {
    if (!orgId) {
      throw new Error("Missing organization context.")
    }
    const priceName = price.name.trim()
    const isUsageBased = price.pricing_model === "USAGE_BASED"
    const payload = {
      organization_id: orgId,
      product_id: productId,
      code,
      name: priceName,
      // Usage-based pricing is stored as PER_UNIT in the price record.
      pricing_model: isUsageBased ? "PER_UNIT" : "FLAT",
      billing_mode: isUsageBased ? "METERED" : "LICENSED",
      billing_interval: price.billing_interval,
      billing_interval_count: price.billing_interval_count,
      tax_behavior: "INCLUSIVE",
      aggregate_usage: isUsageBased ? "SUM" : undefined,
      billing_unit: isUsageBased ? "API_CALL" : undefined,
    }
    const response = await admin.post("/prices", payload)
    return response.data?.data
  }

  const createPriceAmounts = async (
    price: PriceDraft,
    priceId: string,
    priceIndex: number
  ) => {
    if (!orgId) {
      throw new Error("Missing organization context.")
    }
    const currency = price.currency.trim()
    if (!currency) {
      form.setError(`prices.${priceIndex}.currency`, { message: "Currency is required." })
      throw new Error("Missing currency.")
    }

    if (price.pricing_model === "FLAT") {
      if (!price.flat.amounts.length) {
        form.setError(`prices.${priceIndex}.flat.amounts`, {
          message: "Add at least one price amount.",
        })
        throw new Error("Missing flat amounts.")
      }
      const payloads = price.flat.amounts.map((amount, amountIndex) => {
        if (amount.unit_amount == null) {
          form.setError(`prices.${priceIndex}.flat.amounts.${amountIndex}.unit_amount`, {
            message: "Unit price is required.",
          })
          throw new Error("Missing unit amount.")
        }
        return {
          organization_id: orgId,
          price_id: priceId,
          meter_id: null,
          currency,
          unit_amount_cents: Math.round(amount.unit_amount),
          effective_from: toISODate(amount.effective_from),
          effective_to: toISODate(amount.effective_to),
        }
      })
      const sorted = payloads.sort((a, b) => {
        const left = a.effective_from ? new Date(a.effective_from).getTime() : 0
        const right = b.effective_from ? new Date(b.effective_from).getTime() : 0
        return left - right
      })
      for (const payload of sorted) {
        await admin.post("/price_amounts", payload)
      }
      return
    }

    if (!price.usage.rates.length) {
      form.setError(`prices.${priceIndex}.usage.rates`, {
        message: "Add at least one meter rate.",
      })
      throw new Error("Missing usage rates.")
    }

    // Meter associations live on PriceAmount so each rate can bind to a meter independently.
    const payloads = price.usage.rates.map((rate, rateIndex) => {
      if (!rate.meter_id) {
        form.setError(`prices.${priceIndex}.usage.rates.${rateIndex}.meter_id`, {
          message: "Meter is required.",
        })
        throw new Error("Missing meter.")
      }
      if (rate.unit_amount == null) {
        form.setError(`prices.${priceIndex}.usage.rates.${rateIndex}.unit_amount`, {
          message: "Unit price is required.",
        })
        throw new Error("Missing unit amount.")
      }
      if (
        rate.minimum_amount != null &&
        rate.maximum_amount != null &&
        rate.maximum_amount < rate.minimum_amount
      ) {
        form.setError(`prices.${priceIndex}.usage.rates.${rateIndex}.maximum_amount`, {
          message: "Maximum charge must be greater than minimum charge.",
        })
        throw new Error("Invalid minimum/maximum.")
      }

      return {
        organization_id: orgId,
        price_id: priceId,
        meter_id: rate.meter_id,
        currency,
        unit_amount_cents: Math.round(rate.unit_amount),
        minimum_amount_cents: rate.minimum_amount == null ? undefined : Math.round(rate.minimum_amount),
        maximum_amount_cents: rate.maximum_amount == null ? undefined : Math.round(rate.maximum_amount),
        metadata: { aggregation: rate.aggregation },
      }
    })

    await Promise.all(payloads.map((payload) => admin.post("/price_amounts", payload)))
  }

  const resolvePriceCodes = (values: CreateProductFormValues) => {
    const productCodeValue = values.product.code.trim()
    if (!productCodeValue) {
      form.setError("product.code", { message: "Product code is required." })
      throw new Error("Missing product code.")
    }

    const codes = values.prices.map((price, index) => {
      const explicit = price.code?.trim() ?? ""
      if (explicit) return explicit
      const base = buildPriceCode(productCodeValue, price.pricing_model)
      const suffix = values.prices.length > 1 ? `-${index + 1}` : ""
      return `${base}${suffix}`
    })

    const seen = new Map<string, number>()
    let hasDuplicate = false
    codes.forEach((code, index) => {
      if (seen.has(code)) {
        hasDuplicate = true
        const firstIndex = seen.get(code) ?? index
        form.setError(`prices.${index}.code`, { message: "Code must be unique." })
        form.setError(`prices.${firstIndex}.code`, { message: "Code must be unique." })
        return
      }
      seen.set(code, index)
    })

    if (hasDuplicate) {
      throw new Error("Duplicate price codes.")
    }

    return codes
  }

  const createRemainingPricing = async (
    values: CreateProductFormValues,
    productId: string,
    onStageChange?: (stage: "price" | "amount", index: number) => void
  ) => {
    const priceCodes = resolvePriceCodes(values)
    const prices = values.prices

    setStep("price", "loading")
    setStep("amount", "loading")

    for (let index = 0; index < prices.length; index += 1) {
      const price = prices[index]
      let priceId = createdPriceIds[index]

      if (!priceId) {
        onStageChange?.("price", index)
        const created = await createPrice(price, productId, priceCodes[index])
        priceId = created?.id ?? null
        if (!priceId) {
          throw new Error("Price created without an ID.")
        }
        setCreatedPriceIds((prev) => ({ ...prev, [index]: priceId }))
      }

      if (!createdAmountIds[index]) {
        onStageChange?.("amount", index)
        await createPriceAmounts(price, priceId, index)
        setCreatedAmountIds((prev) => ({ ...prev, [index]: true }))
      }
    }

    setStep("price", "success")
    setStep("amount", "success")
  }

  const runCreateFlow = async (values: CreateProductFormValues) => {
    if (!canManage) {
      setOrchestrationError({
        kind: "unknown",
        message: "You do not have permission to create products.",
      })
      return
    }
    if (createdProductId || Object.keys(createdPriceIds).length > 0) {
      setOrchestrationError({
        kind: "unknown",
        message: "Setup already started. Use the retry actions or refresh to start over.",
      })
      return
    }

    setOrchestrationError(null)
    setIsSubmitting(true)
    setStepStatus(defaultStepState)

    let productId: string | null = null
    let currentStage: "product" | "price" | "amount" = "product"
    let currentIndex = -1

    try {
      setStep("product", "loading")
      const product = await createProduct(values)
      productId = product?.id ?? null
      setCreatedProductId(productId)
      setStep("product", "success")

      if (!productId) {
        throw new Error("Product created without an ID.")
      }

      currentStage = "price"
      currentIndex = 0
      await createRemainingPricing(values, productId, (stage, index) => {
        currentStage = stage
        currentIndex = index
      })

      handleNavigateToProduct(productId)
    } catch (err) {
      const detail = getErrorMessage(err, "Something went wrong.")
      const stage = currentStage as "product" | "price" | "amount"
      const priceLabel =
        currentIndex >= 0
          ? values.prices[currentIndex]?.name?.trim() || `Price #${currentIndex + 1}`
          : null

      if (productId && stage === "price") {
        setStep("price", "error")
        setOrchestrationError({
          kind: "product-price",
          message: priceLabel
            ? `${priceLabel} could not be created.`
            : "Product was created, but pricing setup failed.",
          detail,
        })
      } else if (productId && stage === "amount") {
        setStep("amount", "error")
        setOrchestrationError({
          kind: "price-amount",
          message: priceLabel
            ? `${priceLabel} was created, but amount setup failed.`
            : "Price was created, but amount setup failed.",
          detail,
        })
      } else {
        setStep("product", "error")
        setOrchestrationError({
          kind: "unknown",
          message: "Unable to create product.",
          detail,
        })
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  if (!canManage) {
    return <ForbiddenState description="You do not have access to create products." />
  }

  const retryPricing = async () => {
    if (!createdProductId) return
    const values = form.getValues()
    setIsSubmitting(true)
    setOrchestrationError(null)

    try {
      await createRemainingPricing(values, createdProductId)
      handleNavigateToProduct(createdProductId)
    } catch (err) {
      const detail = getErrorMessage(err, "Something went wrong.")
      setStep("price", "error")
      setOrchestrationError({
        kind: "product-price",
        message: "Pricing setup failed.",
        detail,
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  const retryAmount = async () => {
    if (!createdProductId) return
    const values = form.getValues()
    setIsSubmitting(true)
    setOrchestrationError(null)

    try {
      await createRemainingPricing(values, createdProductId)
      handleNavigateToProduct(createdProductId)
    } catch (err) {
      const detail = getErrorMessage(err, "Something went wrong.")
      setStep("amount", "error")
      setOrchestrationError({
        kind: "price-amount",
        message: "Amount setup failed.",
        detail,
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  const submitDisabled = isSubmitting || createdProductId !== null

  return (
    <div className="space-y-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold">Create product</h1>
        <p className="text-text-muted text-sm">
          Set up a product with one or more prices and amounts in one flow.
        </p>
      </div>

      {orchestrationError && (
        <Alert variant="destructive">
          <AlertTitle>{orchestrationError.message}</AlertTitle>
          <AlertDescription>
            {orchestrationError.detail && <p>{orchestrationError.detail}</p>}
            <div className="flex flex-wrap gap-2 pt-2">
              {orchestrationError.kind === "product-price" && (
                <>
                  <Button size="sm" onClick={retryPricing} disabled={isSubmitting}>
                    {isSubmitting ? "Retrying..." : "Retry pricing"}
                  </Button>
                  {createdProductId && (
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => handleNavigateToProduct(createdProductId)}
                    >
                      Go to product detail
                    </Button>
                  )}
                </>
              )}
              {orchestrationError.kind === "price-amount" && (
                <Button size="sm" onClick={retryAmount} disabled={isSubmitting}>
                  {isSubmitting ? "Retrying..." : "Retry amount"}
                </Button>
              )}
            </div>
          </AlertDescription>
        </Alert>
      )}

      <Form {...form}>
        <form onSubmit={form.handleSubmit(runCreateFlow)} className="space-y-6">
          <Card>
            <CardHeader className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
              <div className="space-y-1">
                <CardTitle>Step 1 - Product information</CardTitle>
                <CardDescription>Describe the product that customers will purchase.</CardDescription>
              </div>
              <StepBadge status={stepStatus.product} />
            </CardHeader>
            <CardContent className="grid gap-4 md:grid-cols-2">
              <FormField
                control={form.control}
                name="product.name"
                rules={{ required: "Product name is required." }}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Name</FormLabel>
                    <FormControl>
                      <Input data-testid="product-name" placeholder="Starter plan" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="product.code"
                rules={{ required: "Product code is required." }}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Code</FormLabel>
                    <FormControl>
                      <Input data-testid="product-code" placeholder="starter-plan" {...field} />
                    </FormControl>
                    <FormHint>Used to generate a stable price code.</FormHint>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className="md:col-span-2">
                <FormField
                  control={form.control}
                  name="product.description"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Description</FormLabel>
                      <FormControl>
                        <Textarea
                          data-testid="product-description"
                          placeholder="Optional description for internal teams."
                          rows={3}
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <FormField
                control={form.control}
                name="product.active"
                render={({ field }) => (
                  <FormItem className="flex items-center justify-between rounded-lg border p-3">
                    <div className="space-y-1">
                      <FormLabel>Active</FormLabel>
                      <FormHint>Disable to hide from checkout.</FormHint>
                    </div>
                    <FormControl>
                      <Switch checked={field.value} onCheckedChange={field.onChange} />
                    </FormControl>
                  </FormItem>
                )}
              />
              <div className="md:col-span-2 space-y-3">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium">Metadata</p>
                    <p className="text-text-muted text-xs">
                      Attach structured notes to the product.
                    </p>
                  </div>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    data-testid="product-add-metadata"
                    onClick={() => metadataFields.append({ key: "", value: "" })}
                  >
                    Add metadata
                  </Button>
                </div>
                <div className="space-y-2">
                  {metadataFields.fields.map((field, index) => (
                    <div key={field.id} className="grid gap-2 md:grid-cols-[1fr_1fr_auto]">
                      <FormField
                        control={form.control}
                        name={`product.metadata.${index}.key`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel className="sr-only">Key</FormLabel>
                            <FormControl>
                              <Input
                                data-testid={`product-metadata-key-${index}`}
                                placeholder="key"
                                {...field}
                              />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name={`product.metadata.${index}.value`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel className="sr-only">Value</FormLabel>
                            <FormControl>
                              <Input
                                data-testid={`product-metadata-value-${index}`}
                                placeholder="value"
                                {...field}
                              />
                            </FormControl>
                          </FormItem>
                        )}
                      />
                      <Button
                        type="button"
                        size="icon"
                        variant="ghost"
                        className="md:mt-1"
                        data-testid={`product-metadata-remove-${index}`}
                        onClick={() => metadataFields.remove(index)}
                        disabled={metadataFields.fields.length === 1}
                      >
                        Remove
                      </Button>
                    </div>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
              <div className="space-y-1">
                <CardTitle>Step 2 - Prices</CardTitle>
                <CardDescription>
                  Add one or more prices with their amount schedules.
                </CardDescription>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={() => priceFields.append(buildDefaultPrice())}
                  disabled={submitDisabled}
                >
                  Add price
                </Button>
                <StepBadge status={stepStatus.price} />
              </div>
            </CardHeader>
            <CardContent className="space-y-6">
              {priceFields.fields.map((priceField, index) => (
                <PriceEditor
                  key={priceField.id}
                  index={index}
                  total={priceFields.fields.length}
                  control={form.control}
                  productCode={productCode}
                  meters={meters}
                  metersLoading={metersLoading}
                  metersError={metersError}
                  submitDisabled={submitDisabled}
                  amountStatus={stepStatus.amount}
                  onRemove={() => priceFields.remove(index)}
                />
              ))}
            </CardContent>
          </Card>

          <div className="flex flex-wrap items-center gap-3">
            <Button type="submit" data-testid="product-submit" disabled={submitDisabled}>
              {isSubmitting ? "Creating..." : "Create product"}
            </Button>
            <p className="text-text-muted text-xs">
              Each step is committed separately. Partial failures can be retried.
            </p>
          </div>
        </form>
      </Form>
    </div>
  )
}

function PriceEditor({
  index,
  total,
  control,
  productCode,
  meters,
  metersLoading,
  metersError,
  submitDisabled,
  amountStatus,
  onRemove,
}: {
  index: number
  total: number
  control: Control<CreateProductFormValues>
  productCode: string
  meters: MeterOption[]
  metersLoading: boolean
  metersError: string | null
  submitDisabled: boolean
  amountStatus: StepStatus
  onRemove: () => void
}) {
  const pricingModel = useWatch({
    control,
    name: `prices.${index}.pricing_model`,
  }) as PricingModel
  const priceName = useWatch({
    control,
    name: `prices.${index}.name`,
  }) as string | undefined
  const priceCode = useWatch({
    control,
    name: `prices.${index}.code`,
  }) as string | undefined

  const isUsageBased = pricingModel === "USAGE_BASED"
  const usageFields = useFieldArray({
    control,
    name: `prices.${index}.usage.rates`,
  })
  const flatAmountFields = useFieldArray({
    control,
    name: `prices.${index}.flat.amounts`,
  })

  const trimmedCode = productCode.trim()
  const baseCode = trimmedCode
    ? buildPriceCode(trimmedCode, pricingModel)
    : "auto-generated after product code"
  const suggestedCode = trimmedCode && total > 1 ? `${baseCode}-${index + 1}` : baseCode
  const codePreview = priceCode?.trim() ? priceCode.trim() : suggestedCode
  const title = priceName?.trim() ? priceName.trim() : `Price ${index + 1}`

  return (
    <div className="space-y-6 rounded-lg border p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="space-y-1">
          <p className="text-sm font-medium">{title}</p>
          <p className="text-text-muted text-xs">Configure pricing and amounts.</p>
        </div>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          onClick={onRemove}
          disabled={submitDisabled || total === 1}
        >
          Remove price
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <FormField
          control={control}
          name={`prices.${index}.pricing_model`}
          rules={{ required: "Pricing model is required." }}
          render={({ field }) => (
            <FormItem>
              <FormLabel>Pricing model</FormLabel>
              <Select value={field.value} onValueChange={field.onChange}>
                <FormControl>
                  <SelectTrigger data-testid={`price-${index}-pricing-model`}>
                    <SelectValue placeholder="Select a pricing model" />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  {pricingModelOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name={`prices.${index}.name`}
          rules={{ required: "Price name is required." }}
          render={({ field }) => (
            <FormItem>
              <FormLabel>Price name</FormLabel>
              <FormControl>
                <Input
                  data-testid={`price-${index}-name`}
                  placeholder="Starter monthly"
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name={`prices.${index}.code`}
          render={({ field }) => (
            <FormItem>
              <FormLabel>Price code</FormLabel>
              <FormControl>
                <Input
                  data-testid={`price-${index}-code`}
                  placeholder={suggestedCode}
                  {...field}
                />
              </FormControl>
              <FormHint>Leave blank to auto-generate from the product code.</FormHint>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name={`prices.${index}.currency`}
          rules={{ required: "Currency is required." }}
          render={({ field }) => (
            <FormItem>
              <FormLabel>Currency</FormLabel>
              <FormControl>
                <Input data-testid={`price-${index}-currency`} placeholder="USD" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name={`prices.${index}.billing_interval`}
          rules={{ required: "Billing interval is required." }}
          render={({ field }) => (
            <FormItem>
              <FormLabel>Billing interval</FormLabel>
              <Select value={field.value} onValueChange={field.onChange}>
                <FormControl>
                  <SelectTrigger>
                    <SelectValue placeholder="Select interval" />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  {billingIntervalOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name={`prices.${index}.billing_interval_count`}
          rules={{
            required: "Interval count is required.",
            min: { value: 1, message: "Must be at least 1." },
          }}
          render={({ field }) => (
            <FormItem>
              <FormLabel>Interval count</FormLabel>
              <FormControl>
                <Input
                  type="number"
                  min={1}
                  value={field.value}
                  onChange={(event) => {
                    const parsed = Number(event.target.value)
                    field.onChange(Number.isNaN(parsed) ? 1 : parsed)
                  }}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      </div>

      <div className="rounded-lg border bg-bg-subtle/30 p-3 text-xs text-text-muted">
        Price code preview: <span className="font-medium text-text-primary">{codePreview}</span>
      </div>

      <div className="space-y-4 rounded-lg border p-4">
        <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
          <div className="space-y-1">
            <p className="text-sm font-medium">
              {isUsageBased ? "Usage rates" : "Price amounts"}
            </p>
            <p className="text-text-muted text-xs">
              {isUsageBased
                ? "This price is charged based on usage measured by a meter."
                : "Create one or more scheduled amounts for this price."}
            </p>
          </div>
          <StepBadge status={amountStatus} />
        </div>

        {isUsageBased ? (
          <div className="space-y-4">
            {metersLoading && <p className="text-text-muted text-sm">Loading meters...</p>}
            {metersError && <Alert variant="destructive">{metersError}</Alert>}
            {!metersLoading && !metersError && meters.length === 0 && (
              <p className="text-text-muted text-sm">No meters found. Create a meter first.</p>
            )}
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="text-text-muted text-xs">
                Add one or more meter rates for usage-based pricing.
              </p>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() =>
                  usageFields.append({
                    meter_id: "",
                    unit_amount: null,
                    minimum_amount: null,
                    maximum_amount: null,
                    aggregation: "SUM",
                  })
                }
                disabled={submitDisabled || metersLoading || meters.length === 0}
              >
                Add meter rate
              </Button>
            </div>
            <div className="space-y-3">
              {usageFields.fields.map((rate, rateIndex) => (
                <div key={rate.id} className="grid gap-4 rounded-lg border p-4 md:grid-cols-2">
                  <FormField
                    control={control}
                    name={`prices.${index}.usage.rates.${rateIndex}.meter_id`}
                    rules={{ required: isUsageBased ? "Meter is required." : false }}
                    render={({ field }) => (
                      <FormItem className="md:col-span-2">
                        <FormLabel>Meter</FormLabel>
                        <Select value={field.value} onValueChange={field.onChange}>
                          <FormControl>
                            <SelectTrigger data-testid={`price-${index}-meter-${rateIndex}`}>
                              <SelectValue placeholder="Select a meter" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            {meters.map((meter) => (
                              <SelectItem key={meter.id} value={meter.id}>
                                {meter.name || meter.code || meter.id}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={control}
                    name={`prices.${index}.usage.rates.${rateIndex}.unit_amount`}
                    rules={{
                      required: isUsageBased ? "Unit price is required." : false,
                      min: { value: 0, message: "Amount cannot be negative." },
                    }}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Unit price (minor unit)</FormLabel>
                        <FormControl>
                          <Input
                            data-testid={`price-${index}-unit-amount-${rateIndex}`}
                            type="number"
                            min={0}
                            placeholder="200"
                            value={field.value ?? ""}
                            onChange={(event) => field.onChange(toNumberOrNull(event.target.value))}
                          />
                        </FormControl>
                        <FormHint>Example: 200 = $2.00 per unit</FormHint>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={control}
                    name={`prices.${index}.usage.rates.${rateIndex}.aggregation`}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Aggregation</FormLabel>
                        <Select value={field.value} onValueChange={field.onChange}>
                          <FormControl>
                            <SelectTrigger data-testid={`price-${index}-aggregation-${rateIndex}`}>
                              <SelectValue placeholder="Select aggregation" />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            {aggregationOptions.map((option) => (
                              <SelectItem key={option.value} value={option.value}>
                                {option.label}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={control}
                    name={`prices.${index}.usage.rates.${rateIndex}.minimum_amount`}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Minimum charge (optional)</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            min={0}
                            placeholder="0"
                            value={field.value ?? ""}
                            onChange={(event) => field.onChange(toNumberOrNull(event.target.value))}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={control}
                    name={`prices.${index}.usage.rates.${rateIndex}.maximum_amount`}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Maximum charge (optional)</FormLabel>
                        <FormControl>
                          <Input
                            type="number"
                            min={0}
                            placeholder="10000"
                            value={field.value ?? ""}
                            onChange={(event) => field.onChange(toNumberOrNull(event.target.value))}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <div className="md:col-span-2 flex justify-end">
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      onClick={() => usageFields.remove(rateIndex)}
                      disabled={submitDisabled || usageFields.fields.length === 1}
                    >
                      Remove rate
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="text-text-muted text-xs">
                Add one or more price amounts. Effective dates define future changes.
              </p>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() =>
                  flatAmountFields.append({
                    unit_amount: null,
                    effective_from: "",
                    effective_to: "",
                  })
                }
                disabled={submitDisabled}
              >
                Add amount
              </Button>
            </div>
            <div className="space-y-3">
              {flatAmountFields.fields.map((amount, amountIndex) => (
                <div key={amount.id} className="grid gap-4 rounded-lg border p-4 md:grid-cols-2">
                  <FormField
                    control={control}
                    name={`prices.${index}.flat.amounts.${amountIndex}.unit_amount`}
                    rules={{
                      required: pricingModel === "FLAT" ? "Unit price is required." : false,
                      min: { value: 0, message: "Amount cannot be negative." },
                    }}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Unit price (minor unit)</FormLabel>
                        <FormControl>
                          <Input
                            data-testid={`price-${index}-flat-amount-${amountIndex}`}
                            type="number"
                            min={0}
                            placeholder="5000"
                            value={field.value ?? ""}
                            onChange={(event) => field.onChange(toNumberOrNull(event.target.value))}
                          />
                        </FormControl>
                        <FormHint>Example: 5000 = $50.00</FormHint>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={control}
                    name={`prices.${index}.flat.amounts.${amountIndex}.effective_from`}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Effective from (optional)</FormLabel>
                        <FormControl>
                          <Input type="datetime-local" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={control}
                    name={`prices.${index}.flat.amounts.${amountIndex}.effective_to`}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Effective to (optional)</FormLabel>
                        <FormControl>
                          <Input type="datetime-local" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <div className="md:col-span-2 flex justify-end">
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      onClick={() => flatAmountFields.remove(amountIndex)}
                      disabled={submitDisabled || flatAmountFields.fields.length === 1}
                    >
                      Remove amount
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function StepBadge({
  status,
  labelOverride,
}: {
  status: StepStatus
  labelOverride?: string
}) {
  const { label, variant } = formatStepStatus(status)
  return (
    <Badge variant={variant} className="gap-1">
      {status === "loading" && <Spinner className="size-3" />}
      {labelOverride ?? label}
    </Badge>
  )
}
