<script setup lang="ts">
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
} from "@/components/ui/form";
import { toTypedSchema } from "@vee-validate/zod";
import { useForm } from "vee-validate";
import { ref } from "vue";
import * as z from "zod";

const props = defineProps<{
  open?: boolean;
  trigger?: string;
}>();

const emit = defineEmits<{
  "update:open": [value: boolean];
  submit: [data: { name: string; description: string }];
}>();

const formSchema = toTypedSchema(
  z.object({
    name: z
      .string()
      .min(2, {
        message: "Location name must be at least 2 characters",
      })
      .max(50, {
        message: "Location name must not exceed 50 characters",
      }),
    description: z.string().optional(),
  })
);

const form = useForm({
  validationSchema: formSchema,
  initialValues: {
    name: "",
    description: "",
  },
});

const localOpen = ref(props.open || false);

function onOpenChange(value: boolean) {
  localOpen.value = value;
  emit("update:open", value);

  if (!value) {
    form.resetForm();
  }
}

watch(
  () => props.open,
  (newValue) => {
    if (newValue !== undefined) {
      localOpen.value = newValue;
    }
  }
);

const onSubmit = form.handleSubmit((values) => {
  emit("submit", {
    name: values.name,
    description: values.description || "",
  });

  onOpenChange(false);
});
</script>

<template>
  <Dialog :open="localOpen" @update:open="onOpenChange">
    <DialogTrigger v-if="trigger" as-child>
      <Button>{{ trigger }}</Button>
    </DialogTrigger>

    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>Add New Location</DialogTitle>
        <DialogDescription>
          Enter the details for the new location. Click save when you're done.
        </DialogDescription>
      </DialogHeader>

      <form @submit="onSubmit" class="space-y-4 pt-4">
        <FormField v-slot="{ componentField }" name="name">
          <FormItem>
            <FormLabel>Location Name</FormLabel>
            <FormControl>
              <Input
                v-bind="componentField"
                placeholder="Enter location name"
              />
            </FormControl>
            <FormDescription>
              Name for the location to be added
            </FormDescription>
          </FormItem>
        </FormField>

        <FormField v-slot="{ componentField }" name="description">
          <FormItem>
            <FormLabel>Description</FormLabel>
            <FormControl>
              <Textarea
                v-bind="componentField"
                placeholder="Location description (optional)"
              />
            </FormControl>
            <FormDescription>
              Brief description about this location
            </FormDescription>
          </FormItem>
        </FormField>

        <DialogFooter>
          <Button type="button" variant="outline" @click="onOpenChange(false)">
            Cancel
          </Button>
          <Button type="submit"> Save </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>
