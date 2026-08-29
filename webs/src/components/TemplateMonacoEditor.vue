<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api'
import 'monaco-editor/esm/vs/basic-languages/yaml/yaml.contribution'
import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import 'monaco-editor/min/vs/editor/editor.main.css'

const props = defineProps<{ modelValue: string; language?: string; errors?: string[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const host = ref<HTMLElement>()
let editor: monaco.editor.IStandaloneCodeEditor | undefined
let resizeObserver: ResizeObserver | undefined
let updating = false

;(self as any).MonacoEnvironment = { getWorker: () => new EditorWorker() }

const revealLine = (line: number) => {
  editor?.revealLineInCenter(line)
  editor?.setPosition({ lineNumber: line, column: 1 })
  editor?.focus()
}
defineExpose({ revealLine })

onMounted(() => {
  if (!host.value) return
  editor = monaco.editor.create(host.value, {
    value: props.modelValue,
    language: props.language || 'yaml',
    automaticLayout: false,
    minimap: { enabled: true },
    fontSize: 13,
    lineHeight: 21,
    smoothScrolling: true,
    scrollBeyondLastLine: false,
    wordWrap: 'on',
    padding: { top: 12 },
    theme: document.documentElement.classList.contains('dark') ? 'vs-dark' : 'vs',
  })
  editor.onDidChangeModelContent(() => {
    if (updating) return
    emit('update:modelValue', editor?.getValue() || '')
  })
  resizeObserver = new ResizeObserver(() => editor?.layout())
  resizeObserver.observe(host.value)
})

watch(() => props.modelValue, value => {
  if (!editor || editor.getValue() === value) return
  updating = true; editor.setValue(value); updating = false
})

watch(() => props.language, value => {
  const model = editor?.getModel(); if (model) monaco.editor.setModelLanguage(model, value || 'yaml')
})

watch(() => props.errors, errors => {
  const model = editor?.getModel(); if (!model) return
  monaco.editor.setModelMarkers(model, 'server-validation', (errors || []).map(message => {
    const match = message.match(/line (\d+)/i); const line = match ? Number(match[1]) : 1
    return { severity: monaco.MarkerSeverity.Error, message, startLineNumber: line, startColumn: 1, endLineNumber: line, endColumn: model.getLineMaxColumn(line) }
  }))
})

onBeforeUnmount(() => { resizeObserver?.disconnect(); editor?.dispose() })
</script>

<template><div ref="host" class="monaco-host" /></template>

<style scoped>.monaco-host { width: 100%; height: 100%; min-height: 520px; border: 1px solid var(--el-border-color); border-radius: 8px; overflow: hidden; }</style>
