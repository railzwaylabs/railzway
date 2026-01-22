import { useMemo, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import { admin } from "@/api/client"
import { ForbiddenState } from "@/components/forbidden-state"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Spinner } from "@/components/ui/spinner"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Textarea } from "@/components/ui/textarea"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"

import { AdminCatalogTabs } from "@/features/admin/catalog/components/AdminCatalogTabs"
import { getErrorMessage, isForbiddenError } from "@/lib/api-errors"

type Feature = {
  id: string
  code: string
  name: string
  feature_type: string
  description?: string | null
  active: boolean
  used_by_products?: number | string
  product_count?: number
  products_count?: number
}

const fetchFeatures = async () => {
  const response = await admin.get("/features")
  const payload = response.data?.data
  if (Array.isArray(payload)) {
    return payload as Feature[]
  }
  if (payload && typeof payload === "object") {
    const list = (payload as { features?: Feature[] }).features
    return Array.isArray(list) ? list : []
  }
  return []
}

const formatFeatureType = (value: string) => {
  const normalized = value.trim().toLowerCase()
  if (normalized === "boolean") return "Boolean"
  if (normalized === "metered") return "Usage-based"
  return value || "-"
}

const formatUsageIndicator = (feature: Feature) => {
  const count =
    feature.used_by_products ??
    feature.product_count ??
    feature.products_count
  if (typeof count === "number") return count.toString()
  if (typeof count === "string" && count.trim().length > 0) return count
  return "-"
}

export default function AdminFeaturesPage() {
  const queryClient = useQueryClient()
  const { data, isLoading, error } = useQuery({
    queryKey: ["admin-features"],
    queryFn: fetchFeatures,
  })

  const features = useMemo(() => data ?? [], [data])
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [createName, setCreateName] = useState("")
  const [createCode, setCreateCode] = useState("")
  const [createType, setCreateType] = useState("boolean")
  const [createDescription, setCreateDescription] = useState("")
  const [createError, setCreateError] = useState<string | null>(null)
  const [editingFeature, setEditingFeature] = useState<Feature | null>(null)
  const [editName, setEditName] = useState("")
  const [editDescription, setEditDescription] = useState("")
  const [editError, setEditError] = useState<string | null>(null)
  const [archivingFeature, setArchivingFeature] = useState<Feature | null>(null)
  const [archiveError, setArchiveError] = useState<string | null>(null)

  const createMutation = useMutation({
    mutationFn: async () =>
      admin.post("/features", {
        name: createName.trim(),
        code: createCode.trim(),
        feature_type: createType,
        description: createDescription.trim() || null,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin-features"] })
      setCreateName("")
      setCreateCode("")
      setCreateType("boolean")
      setCreateDescription("")
      setCreateError(null)
      setIsCreateOpen(false)
    },
    onError: (err: any) => {
      setCreateError(getErrorMessage(err, "Unable to create feature."))
    },
  })

  const updateMutation = useMutation({
    mutationFn: async (payload: { id: string; name: string; description: string }) =>
      admin.patch(`/features/${payload.id}`, {
        name: payload.name,
        description: payload.description,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin-features"] })
      setEditingFeature(null)
      setEditError(null)
    },
    onError: (err: any) => {
      setEditError(getErrorMessage(err, "Unable to update feature."))
    },
  })

  const archiveMutation = useMutation({
    mutationFn: async (id: string) => admin.post(`/features/${id}/archive`),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin-features"] })
      setArchivingFeature(null)
      setArchiveError(null)
    },
    onError: (err: any) => {
      setArchiveError(getErrorMessage(err, "Unable to archive feature."))
    },
  })

  if (isForbiddenError(error)) {
    return <ForbiddenState description="You do not have access to features." />
  }

  const isCreateDisabled =
    !createName.trim() || !createCode.trim() || createMutation.isPending

  return (
    <div className="space-y-6">
      <div className="space-y-3">
        <AdminCatalogTabs />
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold">Features</h1>
          <p className="text-text-muted text-sm">
            Features represent monetizable capabilities that customers receive as part of a subscription.
          </p>
          <p className="text-text-muted text-sm">
            Features are used to create customer entitlements and may appear on invoices.
          </p>
        </div>
      </div>

      <Card>
        <CardHeader>
          <div className="space-y-1">
            <CardTitle>Feature catalog</CardTitle>
            <CardDescription>Define the entitlements customers can purchase.</CardDescription>
          </div>
          <CardAction>
            <Dialog
              open={isCreateOpen}
              onOpenChange={(open) => {
                setIsCreateOpen(open)
                if (!open) {
                  setCreateError(null)
                }
              }}
            >
              <DialogTrigger asChild>
                <Button size="sm">Create feature</Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-xl">
                <DialogHeader>
                  <DialogTitle>Create feature</DialogTitle>
                  <DialogDescription>
                    Define a customer-facing capability and how it should appear in contracts.
                  </DialogDescription>
                </DialogHeader>
                <form
                  className="space-y-4"
                  onSubmit={(event) => {
                    event.preventDefault()
                    setCreateError(null)
                    if (isCreateDisabled) return
                    createMutation.mutate()
                  }}
                >
                  {createError && (
                    <Alert variant="destructive">
                      <AlertTitle>Unable to save</AlertTitle>
                      <AlertDescription>{createError}</AlertDescription>
                    </Alert>
                  )}
                  <Alert>
                    <AlertTitle>Immutable feature codes</AlertTitle>
                    <AlertDescription>
                      Feature codes are customer-facing and cannot be changed later.
                    </AlertDescription>
                  </Alert>
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="space-y-2">
                      <Label htmlFor="feature-name">Display name</Label>
                      <Input
                        id="feature-name"
                        value={createName}
                        onChange={(event) => setCreateName(event.target.value)}
                        placeholder="Premium support"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="feature-code">Code</Label>
                      <Input
                        id="feature-code"
                        value={createCode}
                        onChange={(event) => setCreateCode(event.target.value)}
                        placeholder="premium_support"
                      />
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="feature-type">Type</Label>
                    <Select value={createType} onValueChange={setCreateType}>
                      <SelectTrigger id="feature-type">
                        <SelectValue placeholder="Select feature type" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="boolean">Boolean entitlement</SelectItem>
                        <SelectItem value="metered">Usage-based entitlement</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="feature-description">Description</Label>
                    <Textarea
                      id="feature-description"
                      value={createDescription}
                      onChange={(event) => setCreateDescription(event.target.value)}
                      placeholder="Appears on invoices and entitlement summaries."
                    />
                  </div>
                  <DialogFooter className="gap-2">
                    <Button
                      type="button"
                      variant="ghost"
                      onClick={() => setIsCreateOpen(false)}
                      disabled={createMutation.isPending}
                    >
                      Cancel
                    </Button>
                    <Button type="submit" disabled={isCreateDisabled}>
                      {createMutation.isPending ? "Saving..." : "Create feature"}
                    </Button>
                  </DialogFooter>
                </form>
              </DialogContent>
            </Dialog>
          </CardAction>
        </CardHeader>
        <CardContent>
          {isLoading && (
            <div className="flex items-center gap-2 text-text-muted text-sm">
              <Spinner />
              Loading features
            </div>
          )}
          {error && (
            <div className="text-status-error text-sm">
              {getErrorMessage(error, "Unable to load features.")}
            </div>
          )}
          {!isLoading && !error && features.length === 0 && (
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">F</EmptyMedia>
                <EmptyTitle>No features yet</EmptyTitle>
                <EmptyDescription>
                  Create a feature to start defining customer entitlements.
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          )}
          {!isLoading && !error && features.length > 0 && (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Feature name</TableHead>
                  <TableHead>Code</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Used by products</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {features.map((feature) => (
                  <TableRow key={feature.id}>
                    <TableCell className="font-medium">{feature.name}</TableCell>
                    <TableCell className="text-text-muted">{feature.code}</TableCell>
                    <TableCell>
                      <Badge variant="outline">
                        {formatFeatureType(feature.feature_type)}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={feature.active ? "secondary" : "outline"}
                        className={
                          feature.active
                            ? "border-status-success/30 bg-status-success/10 text-status-success"
                            : "text-text-muted"
                        }
                      >
                        {feature.active ? "Active" : "Archived"}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-text-muted">
                      {formatUsageIndicator(feature)}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-2">
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => {
                            setEditingFeature(feature)
                            setEditName(feature.name ?? "")
                            setEditDescription(feature.description ?? "")
                            setEditError(null)
                          }}
                        >
                          Edit
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => {
                            setArchivingFeature(feature)
                            setArchiveError(null)
                          }}
                        >
                          Archive
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog
        open={Boolean(editingFeature)}
        onOpenChange={(open) => {
          if (!open) {
            setEditingFeature(null)
            setEditError(null)
          }
        }}
      >
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>Edit feature</DialogTitle>
            <DialogDescription>
              Update the feature name or description. Code and type remain immutable.
            </DialogDescription>
          </DialogHeader>
          <form
            className="space-y-4"
            onSubmit={(event) => {
              event.preventDefault()
              if (!editingFeature) return
              setEditError(null)
              updateMutation.mutate({
                id: editingFeature.id,
                name: editName.trim(),
                description: editDescription.trim(),
              })
            }}
          >
            {editError && (
              <Alert variant="destructive">
                <AlertTitle>Unable to save</AlertTitle>
                <AlertDescription>{editError}</AlertDescription>
              </Alert>
            )}
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="feature-edit-name">Display name</Label>
                <Input
                  id="feature-edit-name"
                  value={editName}
                  onChange={(event) => setEditName(event.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="feature-edit-code">Code</Label>
                <Input
                  id="feature-edit-code"
                  value={editingFeature?.code ?? ""}
                  disabled
                />
              </div>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="feature-edit-type">Type</Label>
                <Input
                  id="feature-edit-type"
                  value={formatFeatureType(editingFeature?.feature_type ?? "")}
                  disabled
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="feature-edit-description">Description</Label>
              <Textarea
                id="feature-edit-description"
                value={editDescription}
                onChange={(event) => setEditDescription(event.target.value)}
              />
            </div>
            <DialogFooter className="gap-2">
              <Button
                type="button"
                variant="ghost"
                onClick={() => setEditingFeature(null)}
                disabled={updateMutation.isPending}
              >
                Cancel
              </Button>
              <Button type="submit" disabled={updateMutation.isPending || !editName.trim()}>
                {updateMutation.isPending ? "Saving..." : "Save changes"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={Boolean(archivingFeature)}
        onOpenChange={(open) => {
          if (!open) {
            setArchivingFeature(null)
            setArchiveError(null)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Archive feature</AlertDialogTitle>
            <AlertDialogDescription>
              Archived features cannot be attached to new products. Existing subscriptions remain unaffected.
            </AlertDialogDescription>
          </AlertDialogHeader>
          {archiveError && (
            <Alert variant="destructive">
              <AlertTitle>Unable to archive</AlertTitle>
              <AlertDescription>{archiveError}</AlertDescription>
            </Alert>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={archiveMutation.isPending}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={archiveMutation.isPending || !archivingFeature}
              onClick={() => {
                if (!archivingFeature) return
                archiveMutation.mutate(archivingFeature.id)
              }}
            >
              {archiveMutation.isPending ? "Archiving..." : "Archive feature"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
