import { createI18n } from 'vue-i18n'

// Only map languages we can actually render. Unsupported system languages use
// the documented Chinese fallback instead of silently pretending to be English.
const systemLanguage = typeof navigator !== 'undefined' ? navigator.language.toLowerCase() : ''
const systemLocale = systemLanguage.startsWith('zh') ? 'zh' : systemLanguage.startsWith('en') ? 'en' : 'zh'
const savedLocale = typeof localStorage !== 'undefined' ? localStorage.getItem('simpletools-language') : null

const messages = {
  en: {
    appName: 'SimpleTools', tagline: 'Small tools. Clear results.',
    nav: 'Workspace', recent: 'Recent jobs', preferences: 'Preferences',
    convert: 'Image converter', compress: 'Image compressor', pdf: 'PDF to PNG',
    convertDesc: 'Change image formats in a few clicks.', compressDesc: 'Reduce file size while keeping quality.', pdfDesc: 'Render each PDF page as a crisp PNG.',
    addFiles: 'Add files', dropTitle: 'Drop files here', dropHint: 'or choose files from your computer', browse: 'Browse files', folder: 'Choose folder',
    queue: 'File queue', files: 'files', clear: 'Clear all', output: 'Output directory', change: 'Change', start: 'Start processing', processing: 'Processing', done: 'Done', retry: 'Retry', remove: 'Remove',
    noFiles: 'Your queue is empty', noFilesHint: 'Drop files above or use Browse files to get started.',
    format: 'Output format', quality: 'Quality', dpi: 'Resolution', estimate: 'Estimated output',
    ready: 'Ready', queued: 'Queued', failed: 'Failed', cancelled: 'Cancelled', completed: 'completed', allDone: 'All files processed', openFolder: 'Open output folder',
    theme: 'Theme', language: 'Language', light: 'Light', dark: 'Dark', system: 'System', targetBytes: 'Target bytes (0 = off)', pageRange: 'Pages, e.g. 1-3,5', cancel: 'Cancel',
    supported: 'Supports', imageTypes: 'JPG, PNG, WEBP, AVIF', pdfType: 'PDF documents', preserveMetadata: 'Preserve metadata', includeSubfolders: 'Include subfolders', lossless: 'Lossless encoding', updateAvailable: 'Update available', updateNow: 'Install update', updating: 'Installing update', updateFailed: 'Update installation failed', dismiss: 'Dismiss',
    outputPlaceholder: 'Choose a folder for processed files', saved: 'Saved locally',
  },
  zh: {
    appName: 'SimpleTools', tagline: '小工具，清晰结果。', nav: '工作区', recent: '最近任务', preferences: '偏好设置',
    convert: '图片转换', compress: '图片压缩', pdf: 'PDF 转 PNG', convertDesc: '几步即可转换图片格式。', compressDesc: '在保持质量的同时减小文件体积。', pdfDesc: '将每一页 PDF 渲染为清晰 PNG。',
    addFiles: '添加文件', dropTitle: '将文件拖放到这里', dropHint: '或从电脑中选择文件', browse: '选择文件', folder: '选择文件夹', queue: '文件队列', files: '个文件', clear: '清空', output: '输出目录', change: '更改', start: '开始处理', processing: '处理中', done: '完成', retry: '重试', remove: '移除',
    noFiles: '队列为空', noFilesHint: '拖放文件或点击“选择文件”开始。', format: '输出格式', quality: '质量', dpi: '分辨率', estimate: '预计输出', ready: '待处理', queued: '排队中', failed: '失败', cancelled: '已取消', completed: '已完成', allDone: '全部文件处理完成', openFolder: '打开输出目录', preserveMetadata: '保留元数据', includeSubfolders: '包含子文件夹', lossless: '无损编码', updateAvailable: '发现新版本', updateNow: '安装更新', updating: '正在安装更新', updateFailed: '更新安装失败', dismiss: '关闭',
    theme: '主题', language: '语言', light: '浅色', dark: '深色', system: '跟随系统', targetBytes: '目标字节数（0 表示关闭）', pageRange: '页码，例如 1-3,5', cancel: '取消', supported: '支持类型', imageTypes: 'JPG、PNG、WEBP、AVIF', pdfType: 'PDF 文档', outputPlaceholder: '选择处理后文件的目录', saved: '保存在本地',
  },
}

export const i18n = createI18n({ legacy: false, locale: savedLocale === 'zh' || savedLocale === 'en' ? savedLocale : systemLocale, fallbackLocale: 'zh', messages })
