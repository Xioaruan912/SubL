<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api'
import 'monaco-editor/esm/vs/basic-languages/yaml/yaml.contribution'
import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import 'monaco-editor/min/vs/editor/editor.main.css'

;(self as any).MonacoEnvironment = { getWorker: () => new EditorWorker() }

const props = defineProps<{ original: string; modified: string; language?: string }>()
const host = ref<HTMLElement | null>(null)
let editor: monaco.editor.IStandaloneDiffEditor | undefined
let originalModel: monaco.editor.ITextModel | undefined
let modifiedModel: monaco.editor.ITextModel | undefined

function resetModels() {
  if (!editor) return
  originalModel?.dispose()
  modifiedModel?.dispose()
  originalModel = monaco.editor.createModel(props.original || '', props.language || 'yaml')
  modifiedModel = monaco.editor.createModel(props.modified || '', props.language || 'yaml')
  editor.setModel({ original: originalModel, modified: modifiedModel })
}

onMounted(() => {
  if (!host.value) return
  editor = monaco.editor.createDiffEditor(host.value, {
    automaticLayout: true,
    readOnly: true,
    originalEditable: false,
    renderSideBySide: true,
    renderOverviewRuler: true,
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    wordWrap: 'off',
    fontSize: 13,
    lineNumbers: 'on',
  })
  resetModels()
})

watch(() => [props.original, props.modified, props.language], resetModels)

onBeforeUnmount(() => {
  editor?.dispose()
  originalModel?.dispose()
  modifiedModel?.dispose()
})
</script>

<template><div ref="host" class="diff-host" /></template>

<style scoped>
.diff-host { width: 100%; height: 68vh; min-height: 520px; border: 1px solid var(--el-border-color); border-radius: 10px; overflow: hidden; }
</style>
