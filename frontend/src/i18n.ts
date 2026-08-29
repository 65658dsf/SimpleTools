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
    convert: 'Image converter', compress: 'Image compressor', watermark: 'Image watermark', qrcode: 'Text to QR code', pdf: 'PDF to PNG',
    convertDesc: 'Change image formats in a few clicks.', compressDesc: 'Reduce file size while keeping quality.', watermarkDesc: 'Add a styled text watermark while keeping the source format.', qrcodeDesc: 'Generate customizable QR codes or decode QR images locally.', pdfDesc: 'Render each PDF page as a crisp PNG.',
    addFiles: 'Add files', dropTitle: 'Drop files here', dropHint: 'or choose files from your computer', browse: 'Browse files', folder: 'Choose folder',
    queue: 'File queue', files: 'files', clear: 'Clear all', output: 'Output directory', change: 'Change', start: 'Start processing', processing: 'Processing', done: 'Done', retry: 'Retry', remove: 'Remove',
    noFiles: 'Your queue is empty', noFilesHint: 'Drop files above or use Browse files to get started.',
    format: 'Output format', quality: 'Quality', dpi: 'Resolution', estimate: 'Estimated output', estimatedCompressedSize: 'Estimated compressed size', estimatedShort: 'Est.',
    ready: 'Ready', queued: 'Queued', failed: 'Failed', cancelled: 'Cancelled', completed: 'completed', allDone: 'All files processed', openFolder: 'Open output folder',
    theme: 'Theme', language: 'Language', light: 'Light', dark: 'Dark', system: 'System', targetBytes: 'Target size (0 = off)', pageRange: 'Pages, e.g. 1-3,5', cancel: 'Cancel',
    supported: 'Supports', imageTypes: 'JPG, PNG, WEBP, AVIF, ICO, SVG', pdfType: 'PDF documents', preserveMetadata: 'Preserve metadata', includeSubfolders: 'Include subfolders', lossless: 'Lossless encoding', updateAvailable: 'Update available', updateNow: 'Install update', updating: 'Installing update', updateFailed: 'Update installation failed', dismiss: 'Dismiss',
    outputPlaceholder: 'Default: app folder/output (or choose another)', saved: 'Saved locally',
    livePreview: 'Live preview', beforeWatermark: 'Before', afterWatermark: 'After', watermarkPreviewEmpty: 'Select an image to preview', selectPreview: 'Preview this image', compareSlider: 'Before and after comparison', previewing: 'Updating watermark preview', previewFailed: 'Preview could not be generated',
    watermarkText: 'Watermark text', watermarkTextPlaceholder: 'Enter watermark text', watermarkPreset: 'Style preset', presetSubtle: 'Subtle corner', presetCentered: 'Centered', presetDiagonal: 'Diagonal', presetDense: 'Dense repeat',
    watermarkStyle: 'Watermark style', font: 'Font', fontNoto: 'Noto Sans SC', size: 'Size', color: 'Color', opacity: 'Opacity', position: 'Position', rotation: 'Rotation', margin: 'Edge margin', tile: 'Repeat across image', tileSpacing: 'Repeat spacing', shadow: 'Text shadow', applyWatermark: 'Add watermark',
    positionTopLeft: 'Top left', positionTopCenter: 'Top center', positionTopRight: 'Top right', positionCenterLeft: 'Center left', positionCenter: 'Center', positionCenterRight: 'Center right', positionBottomLeft: 'Bottom left', positionBottomCenter: 'Bottom center', positionBottomRight: 'Bottom right', sourceFormat: 'Source format',
    bytes: 'bytes', qrPreviewAlt: 'Generated QR code preview', qrGenerating: 'Generating preview', qrPreviewFailed: 'Preview could not be generated', qrEmptyPreview: 'Enter text to generate a QR code',
    qrGenerateTab: 'Generate QR', qrDecodeTab: 'Decode QR', qrDecodeDesc: 'Read text from a QR code image locally.', qrDecodeDropTitle: 'Drop a QR code image here', qrDecodeDropHint: 'or choose an image from your computer', qrDecodeBrowse: 'Choose image', qrDecodePreview: 'Image preview', qrDecodeEmpty: 'Select a QR code image to decode', qrDecoding: 'Decoding QR code', qrDecodedText: 'Decoded text', qrCopyText: 'Copy text', qrCopied: 'Copied', qrDecodeFailed: 'Could not decode a QR code from this image', qrDecodeNoCode: 'No QR code was found in this image', qrDecodeUnavailable: 'QR decoding is available in the desktop app', qrDecodeImageOnly: 'Please choose an image file',
    qrContentLength: 'Characters', qrByteLength: 'UTF-8 bytes', qrCorrection: 'Error correction', qrSettings: 'QR code settings', qrText: 'Text content', qrTextPlaceholder: 'Enter text, a link, or any message', qrSize: 'Image size',
    qrCorrectionLow: 'Low correction (7%)', qrCorrectionMedium: 'Medium correction (15%)', qrCorrectionQuartile: 'Quartile correction (25%)', qrCorrectionHigh: 'High correction (30%)',
    qrColors: 'Colors', resetColors: 'Reset colors', foreground: 'Foreground', background: 'Background', fileName: 'File name', qrSave: 'Save PNG', qrSaving: 'Saving', qrSaveFailed: 'QR code could not be saved',
  },
  zh: {
    appName: 'SimpleTools', tagline: '小工具，清晰结果。', nav: '工作区', recent: '最近任务', preferences: '偏好设置',
    convert: '图片转换', compress: '图片压缩', watermark: '图片水印', qrcode: '文本生成二维码', pdf: 'PDF 转 PNG', convertDesc: '几步即可转换图片格式。', compressDesc: '在保持质量的同时减小文件体积。', watermarkDesc: '自定义文本与样式，为图片添加水印并保留原格式。', qrcodeDesc: '生成可自定义的二维码，或在本地解析二维码图片。', pdfDesc: '将每一页 PDF 渲染为清晰 PNG。',
    addFiles: '添加文件', dropTitle: '将文件拖放到这里', dropHint: '或从电脑中选择文件', browse: '选择文件', folder: '选择文件夹', queue: '文件队列', files: '个文件', clear: '清空', output: '输出目录', change: '更改', start: '开始处理', processing: '处理中', done: '完成', retry: '重试', remove: '移除',
    noFiles: '队列为空', noFilesHint: '拖放文件或点击“选择文件”开始。', format: '输出格式', quality: '质量', dpi: '分辨率', estimate: '预计输出', estimatedCompressedSize: '预计压缩后大小', estimatedShort: '预计', ready: '待处理', queued: '排队中', failed: '失败', cancelled: '已取消', completed: '已完成', allDone: '全部文件处理完成', openFolder: '打开输出目录', preserveMetadata: '保留元数据', includeSubfolders: '包含子文件夹', lossless: '无损编码', updateAvailable: '发现新版本', updateNow: '安装更新', updating: '正在安装更新', updateFailed: '更新安装失败', dismiss: '关闭',
    theme: '主题', language: '语言', light: '浅色', dark: '深色', system: '跟随系统', targetBytes: '目标大小（0 表示关闭）', pageRange: '页码，例如 1-3,5', cancel: '取消', supported: '支持类型', imageTypes: 'JPG、PNG、WEBP、AVIF、ICO、SVG', pdfType: 'PDF 文档', outputPlaceholder: '默认：软件目录/output，可选择其他目录', saved: '保存在本地',
    livePreview: '实时预览', beforeWatermark: '添加前', afterWatermark: '添加后', watermarkPreviewEmpty: '选择图片后可预览', selectPreview: '预览此图片', compareSlider: '添加水印前后对比', previewing: '正在更新水印预览', previewFailed: '无法生成预览',
    watermarkText: '水印文本', watermarkTextPlaceholder: '输入水印文本', watermarkPreset: '样式预设', presetSubtle: '轻盈角标', presetCentered: '居中强调', presetDiagonal: '斜向平铺', presetDense: '密集平铺',
    watermarkStyle: '水印样式', font: '字体', fontNoto: '思源黑体', size: '字号', color: '颜色', opacity: '不透明度', position: '位置', rotation: '旋转', margin: '边距', tile: '平铺水印', tileSpacing: '平铺间距', shadow: '文字阴影', applyWatermark: '添加水印',
    positionTopLeft: '左上', positionTopCenter: '上中', positionTopRight: '右上', positionCenterLeft: '左中', positionCenter: '居中', positionCenterRight: '右中', positionBottomLeft: '左下', positionBottomCenter: '下中', positionBottomRight: '右下', sourceFormat: '保留原格式',
    bytes: '字节', qrPreviewAlt: '生成的二维码预览', qrGenerating: '正在生成预览', qrPreviewFailed: '无法生成预览', qrEmptyPreview: '输入文本后生成二维码',
    qrGenerateTab: '生成二维码', qrDecodeTab: '解析二维码', qrDecodeDesc: '在本地读取二维码图片中的文本。', qrDecodeDropTitle: '将二维码图片拖放到这里', qrDecodeDropHint: '或从电脑中选择一张图片', qrDecodeBrowse: '选择图片', qrDecodePreview: '图片预览', qrDecodeEmpty: '选择二维码图片开始解析', qrDecoding: '正在解析二维码', qrDecodedText: '解析结果', qrCopyText: '复制文本', qrCopied: '已复制', qrDecodeFailed: '无法从这张图片解析二维码', qrDecodeNoCode: '图片中未找到二维码', qrDecodeUnavailable: '二维码解析仅在桌面应用中可用', qrDecodeImageOnly: '请选择图片文件',
    qrContentLength: '字符数', qrByteLength: 'UTF-8 字节', qrCorrection: '纠错级别', qrSettings: '二维码设置', qrText: '文本内容', qrTextPlaceholder: '输入文本、链接或任意内容', qrSize: '图片尺寸',
    qrCorrectionLow: '低纠错（7%）', qrCorrectionMedium: '中纠错（15%）', qrCorrectionQuartile: '较高纠错（25%）', qrCorrectionHigh: '高纠错（30%）',
    qrColors: '颜色', resetColors: '重置颜色', foreground: '前景色', background: '背景色', fileName: '文件名', qrSave: '保存 PNG', qrSaving: '正在保存', qrSaveFailed: '二维码保存失败',
  },
}

export const i18n = createI18n({ legacy: false, locale: savedLocale === 'zh' || savedLocale === 'en' ? savedLocale : systemLocale, fallbackLocale: 'zh', messages })
