<script setup lang="ts">
import { ref } from "vue";

const props = defineProps<{
  imageSrc: string;
  defaultLines: Line[];
  disable?: boolean;
  direction?: string;
  loading?: boolean;
}>();

export interface Point {
  x: number;
  y: number;
}

export interface Line {
  start: Point;
  end: Point;
  draggingOffset?: Point;
  selected?: boolean;
  color?: string;
  direction?: string;
}

// Array warna untuk digunakan pada garis
const lineColors = [
  "#4B39EF", // Warna biru asli
  "#FF5722", // Oranye
  "#4CAF50", // Hijau
  "#F44336", // Merah
  "#9C27B0", // Ungu
  "#FFEB3B", // Kuning
  "#00BCD4", // Cyan
  "#FF9800", // Oranye tua
  "#795548", // Cokelat
  "#607D8B", // Biru abu-abu
];

const pathData = `
  M13,18.75a.74.74,0,0,1-.53-.22.75.75,0,0,1,0-1.06L17.94,12,12.47,6.53a.75.75,0,0,1,1.06-1.06l6,6a.75.75,0,0,1,0,1.06l-6,6A.74.74,0,0,1,13,18.75Z
  M19,12.75H5a.75.75,0,0,1,0-1.5H19a.75.75,0,0,1,0,1.5Z
`;

const emit = defineEmits<{
  (event: "onChange", lines: Line[]): void;
  (event: "onSelectedLine", line: Line | null): void;
  (event: "initializedCanvas"): void;
}>();

const lines = ref<Line[]>([]);
const imageRef = ref<HTMLImageElement | null>(null);
const canvasWidth = ref(0);
const canvasHeight = ref(0);
const stageSize = ref({ width: 0, height: 0 });

const initializeCanvas = () => {
  if (!imageRef.value) return;
  const { naturalWidth, naturalHeight, clientWidth, clientHeight } =
    imageRef.value;
  const scaleX = clientWidth / naturalWidth;
  const scaleY = clientHeight / naturalHeight;

  canvasWidth.value = clientWidth;
  canvasHeight.value = clientHeight;

  stageSize.value = { width: clientWidth, height: clientHeight };

  lines.value = props.defaultLines.map((line, index) => {
    return {
      ...line,
      start: { x: line.start.x * scaleX, y: line.start.y * scaleY },
      end: { x: line.end.x * scaleX, y: line.end.y * scaleY },
      // Memberikan warna default dari array lineColors
      color: line.color || lineColors[index % lineColors.length],
    };
  });

  emit("initializedCanvas");
};

// Menghitung posisi tengah dan sudut garis
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const calculateArrowProps = (line: Line) => {
  const midX = (line.start.x + line.end.x) / 2;
  const midY = (line.start.y + line.end.y) / 2;
  const dx = line.end.x - line.start.x;
  const dy = line.end.y - line.start.y;
  const angle = Math.atan2(dy, dx) * (180 / Math.PI); // Rotasi panah
  return { midX, midY, angle };
};

// Update posisi titik handle
const updatePoint = (index: number, pointType: "start" | "end", event: any) => {
  const line = lines.value[index];
  line[pointType] = { x: event.target.x(), y: event.target.y() };
};

// Simpan offset saat mulai drag garis
const onDragStart = (index: number, event: any) => {
  const line = lines.value[index];
  const mouseX = event.evt.x;
  const mouseY = event.evt.y;

  const centerX = (line.start.x + line.end.x) / 2;
  const centerY = (line.start.y + line.end.y) / 2;

  line.draggingOffset = {
    x: mouseX - centerX,
    y: mouseY - centerY,
  };

  event.target.getStage().container().style.cursor = "grabbing";
};

// Perbarui posisi garis
const dragLine = (index: number, event: any) => {
  const line = lines.value[index];
  if (!line.draggingOffset) return;

  const mouseX = event.evt.x;
  const mouseY = event.evt.y;

  const dx =
    mouseX -
    line.draggingOffset.x -
    (line.start.x + (line.end.x - line.start.x) / 2);
  const dy =
    mouseY -
    line.draggingOffset.y -
    (line.start.y + (line.end.y - line.start.y) / 2);

  line.start.x += dx;
  line.start.y += dy;
  line.end.x += dx;
  line.end.y += dy;

  // Reset posisi elemen Konva
  event.target.x(0);
  event.target.y(0);
};

// Reset kursor saat drag selesai
const onDragEnd = (event: any) => {
  event.target.getStage().container().style.cursor = "default";
};

// Cursor changes for line and circle (handle)
const onMouseEnterLine = (event: any) => {
  if (!props.disable) event.target.getStage().container().style.cursor = "move";
};

const onMouseEnterCircle = (event: any) => {
  if (!props.disable)
    event.target.getStage().container().style.cursor = "pointer";
};

const onMouseLeave = (event: any) => {
  if (!props.disable)
    event.target.getStage().container().style.cursor = "default";
};

const onDragStartCircle = (event: any) => {
  if (!props.disable)
    event.target.getStage().container().style.cursor = "grabbing";
};

// Fungsi untuk menambah garis
const addLine = (direction: "ltr" | "rtl" | "ttb" | "btt") => {
  const midX = canvasWidth.value / 2;
  const midY = canvasHeight.value / 2;
  const offsetX = canvasWidth.value / 12;
  const offsetY = canvasHeight.value / 8;
  const lineWidth = canvasWidth.value / 3;
  const lineHeight = canvasHeight.value / 3;

  const lineConfig = {
    ltr: {
      start: { x: offsetX, y: midY - lineHeight / 2 },
      end: { x: offsetX, y: midY + lineHeight / 2 },
    },
    rtl: {
      start: { x: canvasWidth.value - offsetX, y: midY + lineHeight / 2 },
      end: { x: canvasWidth.value - offsetX, y: midY - lineHeight / 2 },
    },
    ttb: {
      start: { x: midX + lineWidth / 2, y: offsetY },
      end: { x: midX - lineWidth / 2, y: offsetY },
    },
    btt: {
      start: { x: midX - lineWidth / 2, y: canvasHeight.value - offsetY },
      end: { x: midX + lineWidth / 2, y: canvasHeight.value - offsetY },
    },
  };

  // Menambahkan warna ke garis baru berdasarkan jumlah garis yang ada
  const lineIndex = lines.value.length;
  const lineColor = lineColors[lineIndex % lineColors.length];

  const newLine: Line = {
    ...lineConfig[direction],
    selected: false,
    direction: direction,
    color: lineColor,
  };

  lines.value.push(newLine);
};

// Fungsi untuk memilih garis
const selectLine = (index: number) => {
  if (!imageRef.value) return;

  const { naturalWidth, naturalHeight, clientWidth, clientHeight } =
    imageRef.value;
  const scaleX = naturalWidth / clientWidth;
  const scaleY = naturalHeight / clientHeight;

  lines.value.forEach((line, i) => {
    line.selected = i === index;
  });

  const selectedLine = lines.value.find((line) => line.selected);

  emit(
    "onSelectedLine",
    selectedLine
      ? {
          start: {
            x: selectedLine.start.x * scaleX,
            y: selectedLine.start.y * scaleY,
          },
          end: {
            x: selectedLine.end.x * scaleX,
            y: selectedLine.end.y * scaleY,
          },
          color: selectedLine.color, // Meneruskan warna ke event
        }
      : null
  );
};

// Fungsi untuk menghapus garis yang dipilih
const removeLine = (index: number) => {
  lines.value = lines.value.filter((line, i) => index !== i);
};

const onLinesChange = () => {
  if (!imageRef.value) return;

  const { naturalWidth, naturalHeight, clientWidth, clientHeight } =
    imageRef.value;
  const scaleX = naturalWidth / clientWidth;
  const scaleY = naturalHeight / clientHeight;

  const lineData = lines.value.map((line) => ({
    start: { x: line.start.x * scaleX, y: line.start.y * scaleY },
    end: { x: line.end.x * scaleX, y: line.end.y * scaleY },
    selected: line.selected,
    direction: line.direction,
    color: line.color,
  }));

  emit("onChange", lineData);
};

const directionConfig = (direction: string | undefined) => {
  const configMap: Record<string, { x: number; y: number; rotation: number }> =
    {
      rtl: { x: canvasWidth.value - 30, y: 70, rotation: 180 },
      ttb: { x: canvasWidth.value - 30, y: 25, rotation: 90 },
      btt: { x: canvasWidth.value - 80, y: 70, rotation: 270 },
    };

  return (
    configMap[direction || ""] || {
      x: canvasWidth.value - 80,
      y: 25,
      rotation: 0,
    }
  );
};

// Watch perubahan pada array lines
watch(
  lines,
  () => {
    onLinesChange();
  },
  { deep: true }
);

defineExpose({ addLine, removeLine });
</script>

<template>
  <div class="w-full relative">
    <img
      ref="imageRef"
      :src="imageSrc"
      @load="initializeCanvas"
      alt="Image"
      class="absolute w-full h-auto inset-0 object-cover rounded-md"
    />

    <v-stage :config="stageSize" class="absolute inset-0 rounded-md">
      <v-layer>
        <template v-for="(line, index) in lines" :key="index">
          <!-- Garis -->
          <v-line
            :config="{
              points: [line.start.x, line.start.y, line.end.x, line.end.y],
              stroke: line.color || '#4B39EF', // Menggunakan warna dari objek line
              strokeWidth: 5,
              draggable: !disable,
            }"
            @dragstart="onDragStart(index, $event)"
            @dragmove="dragLine(index, $event)"
            @dragend="onDragEnd($event)"
            @mouseenter="onMouseEnterLine($event)"
            @mouseleave="onMouseLeave($event)"
            @click="selectLine(index)"
          />

          <!-- Panah di tengah garis -->
          <!-- <v-path :config="{
            x: calculateArrowProps(line).midX,
            y: calculateArrowProps(line).midY,
            rotation: calculateArrowProps(line).angle - 90,
            fill: line.color || '#4B39EF',  // Menggunakan warna dari objek line
            data: pathData,
            scaleX: 2,
            scaleY: 2,
            offsetY: 11.9
          }" @click="selectLine(index)" /> -->

          <!-- Titik awal -->
          <v-circle
            :config="{
              x: line.start.x,
              y: line.start.y,
              radius: 3,
              fill: 'red',
              draggable: !disable,
            }"
            @dragmove="updatePoint(index, 'start', $event)"
            @dragstart="onDragStartCircle($event)"
            @dragend="onDragEnd($event)"
            @mouseenter="onMouseEnterCircle($event)"
            @mouseleave="onMouseLeave($event)"
            @click="selectLine(index)"
          />

          <!-- Titik akhir -->
          <v-circle
            :config="{
              x: line.end.x,
              y: line.end.y,
              radius: 3,
              fill: 'blue',
              draggable: !disable,
            }"
            @dragmove="updatePoint(index, 'end', $event)"
            @dragstart="onDragStartCircle($event)"
            @dragend="onDragEnd($event)"
            @mouseenter="onMouseEnterCircle($event)"
            @mouseleave="onMouseLeave($event)"
            @click="selectLine(index)"
          />
        </template>

        <!-- Direction -->
        <v-path
          :config="{
            ...directionConfig(direction),
            fill: '#4B39EF',
            data: pathData,
            scaleX: 2,
            scaleY: 2,
            offsetY: 0,
          }"
        />
      </v-layer>
    </v-stage>

    <div
      v-if="loading"
      class="absolute inset-0 flex items-center justify-center rounded-md"
    >
      <div class="loader"></div>
    </div>
  </div>
</template>
