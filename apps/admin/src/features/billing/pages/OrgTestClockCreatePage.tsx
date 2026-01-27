import { useState } from "react"
import { useNavigate, useParams } from "react-router-dom"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { ArrowLeft } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { Separator } from "@/components/ui/separator"

import { admin } from "@/api/client"
import { getErrorMessage } from "@/lib/api-errors"

const formSchema = z.object({
  name: z.string().min(1, "Name is required"),
  initial_time: z.string().optional(), // ISO datetime string
})

type FormValues = z.infer<typeof formSchema>

export default function OrgTestClockCreatePage() {
  const { orgId } = useParams()
  const navigate = useNavigate()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: "",
      initial_time: new Date().toISOString().slice(0, 16), // datetime-local format
    },
  })

  const onSubmit = async (values: FormValues) => {
    setIsSubmitting(true)
    try {
      const payload = {
        name: values.name,
        initial_time: new Date(values.initial_time || Date.now()).toISOString(),
      }

      await admin.post("/test-clocks", payload)

      navigate(`/orgs/${orgId}/subscriptions?tab=test-clocks`)
    } catch (err) {
      setError(getErrorMessage(err, "Failed to create test clock"))
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={() => navigate(`/orgs/${orgId}/subscriptions?tab=test-clocks`)}
        >
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-2xl font-semibold">New simulation</h1>
          <p className="text-text-muted text-sm">
            Create a test clock to simulate billing scenarios.
          </p>
        </div>
      </div>
      <Separator />

      {error && (
        <Alert variant="destructive">
          <AlertTitle>Error</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-8">
          <FormField
            control={form.control}
            name="name"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Simulation Name</FormLabel>
                <FormControl>
                  <Input placeholder="e.g. Test Scenario A" {...field} />
                </FormControl>
                <FormDescription>
                  A descriptive name for this simulation environment.
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="initial_time"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Start Time</FormLabel>
                <FormControl>
                  <Input type="datetime-local" {...field} />
                </FormControl>
                <FormDescription>
                  The virtual time this clock will start at.
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className="flex justify-end gap-3">
            <Button
              type="button"
              variant="outline"
              disabled={isSubmitting}
              onClick={() => navigate(-1)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? "Creating..." : "Create simulation"}
            </Button>
          </div>
        </form>
      </Form>
    </div>
  )
}
