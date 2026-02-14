import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import * as z from "zod";
import type { CatalogItem } from "../types";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";

interface ConnectionFormProps {
  item: CatalogItem;
  onSubmit: (values: any) => void;
  isSubmitting?: boolean;
}

export function ConnectionForm({ item, onSubmit, isSubmitting }: ConnectionFormProps) {
  // Generate a dynamic Zod schema from the JSON Schema
  const schemaObj: Record<string, any> = {
    name: z.string().min(1, "Connection name is required"),
  };

  const properties = item.schema?.properties || {};
  const required = item.schema?.required || [];

  Object.entries(properties).forEach(([key, prop]: [string, any]) => {
    let fieldSchema = z.string();
    if (required.includes(key)) {
      fieldSchema = fieldSchema.min(1, `${prop.title || key} is required`);
    } else {
      // @ts-ignore
      fieldSchema = fieldSchema.optional();
    }
    schemaObj[key] = fieldSchema;
  });

  const formSchema = z.object(schemaObj);

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: `${item.name} Connection`,
      ...Object.fromEntries(Object.keys(properties).map((k) => [k, ""])),
    },
  });

  const handleSubmit = (values: any) => {
    const { name, ...rest } = values;

    // Distinguish between config and credentials based on auth_type or heuristics
    // For now, simplify: if auth_type is api_key, everything in schema is credential
    // If oauth2, schema is likely config
    const output = {
      name,
      config: item.auth_type === "api_key" ? {} : rest,
      credentials: item.auth_type === "api_key" ? rest : {},
    };

    onSubmit(output);
  };

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-6">
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Connection Name</FormLabel>
              <FormControl>
                <Input placeholder="Acme Productions Slack" {...field} value={(field.value as string) || ""} />
              </FormControl>
              <FormDescription>
                A friendly name to identify this connection.
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <div className="space-y-4 pt-4 border-t">
          <h3 className="text-sm font-medium">Integration Settings</h3>
          {Object.entries(properties).map(([key, prop]: [string, any]) => (
            <FormField
              key={key}
              control={form.control}
              name={key}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{prop.title || key}</FormLabel>
                  <FormControl>
                    {key.includes("secret") || key.includes("url") || key.includes("key") || key.includes("token") ? (
                      <Input
                        type={key.includes("secret") || key.includes("key") || key.includes("token") ? "password" : "text"}
                        placeholder={prop.description || ""}
                        {...field}
                        value={(field.value as string) || ""}
                      />
                    ) : (
                      <Input placeholder={prop.description || ""} {...field} value={(field.value as string) || ""} />
                    )}
                  </FormControl>
                  {prop.description && (
                    <FormDescription>{prop.description}</FormDescription>
                  )}
                  <FormMessage />
                </FormItem>
              )}
            />
          ))}
        </div>

        <Button type="submit" className="w-full" disabled={isSubmitting}>
          {isSubmitting ? "Connecting..." : "Connect Integration"}
        </Button>
      </form>
    </Form>
  );
}
