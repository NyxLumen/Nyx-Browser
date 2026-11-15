<template>
  <component
    v-if="iconComponent"
    :is="iconComponent"
    :size="size"
    :stroke-width="strokeWidth"
    :color="color"
    :class="['nyx-icon', customClass]"
    aria-hidden="true"
  />
  <i
    v-else
    :class="['material-icons', 'nyx-icon--fallback', customClass]"
    aria-hidden="true"
  >{{ name }}</i>
</template>

<script setup lang="ts">
import { computed, type Component } from "vue";
import {
  Folder,
  FolderOpen,
  FileText,
  FileCode,
  FileArchive,
  FileSpreadsheet,
  Image,
  Video,
  Music,
  File,
  X,
  Edit2,
  Trash2,
  Download,
  ExternalLink,
  Info,
  MoreVertical,
  Menu,
  Search,
  Settings,
  List,
  LayoutGrid,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  ChevronUp,
  ArrowLeft,
  ArrowRight,
  RotateCw,
  Check,
  Plus,
  Minus,
  Copy,
  FolderInput,
  Share2,
  HelpCircle,
  AlertTriangle,
  Lock,
  Eye,
  EyeOff,
  Terminal,
  Upload,
  UploadCloud,
  Moon,
  Sun,
  HardDrive,
  User,
  LogOut,
  Maximize2,
  Minimize2,
  ZoomIn,
  ZoomOut,
  SlidersHorizontal,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  Filter,
  CheckSquare,
  Square,
  FileQuestion,
  RefreshCw,
} from "lucide-vue-next";

interface Props {
  name: string;
  size?: number | string;
  strokeWidth?: number | string;
  color?: string;
  customClass?: string;
}

const props = withDefaults(defineProps<Props>(), {
  size: "1.25em",
  strokeWidth: 1.5,
  color: "currentColor",
  customClass: "",
});

const iconMap: Record<string, Component> = {
  // Navigation & Shell
  menu: Menu,
  close: X,
  x: X,
  search: Search,
  settings: Settings,
  more_vert: MoreVertical,
  more_horizontal: MoreVertical,
  chevron_left: ChevronLeft,
  chevron_right: ChevronRight,
  chevron_down: ChevronDown,
  chevron_up: ChevronUp,
  arrow_back: ArrowLeft,
  arrow_forward: ArrowRight,
  arrow_upward: ArrowUp,
  arrow_downward: ArrowDown,
  sort: ArrowUpDown,
  filter: Filter,
  refresh: RefreshCw,
  rotate_right: RotateCw,

  // File Types
  folder: Folder,
  folder_open: FolderOpen,
  insert_drive_file: FileText,
  file: File,
  text_snippet: FileText,
  description: FileText,
  image: Image,
  photo: Image,
  movie: Video,
  video_file: Video,
  videocam: Video,
  audiotrack: Music,
  music_note: Music,
  code: FileCode,
  archive: FileArchive,
  table_chart: FileSpreadsheet,
  csv: FileSpreadsheet,
  help: HelpCircle,
  unknown: FileQuestion,

  // File Actions
  file_download: Download,
  download: Download,
  file_upload: Upload,
  upload: Upload,
  cloud_upload: UploadCloud,
  mode_edit: Edit2,
  edit: Edit2,
  edit_note: Edit2,
  delete: Trash2,
  delete_outline: Trash2,
  content_copy: Copy,
  copy: Copy,
  drive_file_move: FolderInput,
  move: FolderInput,
  share: Share2,
  open_in_new: ExternalLink,
  info: Info,
  info_outline: Info,

  // Status & Controls
  check: Check,
  check_box: CheckSquare,
  check_box_outline_blank: Square,
  add: Plus,
  plus: Plus,
  remove: Minus,
  minus: Minus,
  warning: AlertTriangle,
  lock: Lock,
  visibility: Eye,
  visibility_off: EyeOff,
  terminal: Terminal,
  dark_mode: Moon,
  light_mode: Sun,
  storage: HardDrive,
  person: User,
  account_circle: User,
  exit_to_app: LogOut,
  fullscreen: Maximize2,
  fullscreen_exit: Minimize2,
  zoom_in: ZoomIn,
  zoom_out: ZoomOut,
  tune: SlidersHorizontal,

  // View modes
  view_list: List,
  list: List,
  view_module: LayoutGrid,
  grid: LayoutGrid,
};

const iconComponent = computed<Component | null>(() => {
  const normalized = props.name.trim().toLowerCase();
  return iconMap[normalized] || null;
});
</script>

<style scoped>
.nyx-icon {
  display: inline-block;
  vertical-align: middle;
  flex-shrink: 0;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.nyx-icon--fallback {
  font-size: inherit;
  vertical-align: middle;
}
</style>
