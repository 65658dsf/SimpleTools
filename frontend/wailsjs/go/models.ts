export namespace app {
	
	export class InputFile {
	    path: string;
	    name: string;
	    size: number;
	    kind: string;
	    relativePath?: string;
	
	    static createFrom(source: any = {}) {
	        return new InputFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.size = source["size"];
	        this.kind = source["kind"];
	        this.relativePath = source["relativePath"];
	    }
	}
	export class JobItem {
	    id: string;
	    path: string;
	    name: string;
	    state: string;
	    progress: number;
	    output?: string;
	    outputs?: string[];
	    error?: string;
	    warning?: string;
	
	    static createFrom(source: any = {}) {
	        return new JobItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.name = source["name"];
	        this.state = source["state"];
	        this.progress = source["progress"];
	        this.output = source["output"];
	        this.outputs = source["outputs"];
	        this.error = source["error"];
	        this.warning = source["warning"];
	    }
	}
	export class JobStatus {
	    id: string;
	    state: string;
	    total: number;
	    completed: number;
	    failed: number;
	    progress: number;
	    current?: string;
	    error?: string;
	    outputs?: string[];
	    items: JobItem[];
	    // Go type: time
	    startedAt: any;
	    // Go type: time
	    finishedAt?: any;
	
	    static createFrom(source: any = {}) {
	        return new JobStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.state = source["state"];
	        this.total = source["total"];
	        this.completed = source["completed"];
	        this.failed = source["failed"];
	        this.progress = source["progress"];
	        this.current = source["current"];
	        this.error = source["error"];
	        this.outputs = source["outputs"];
	        this.items = this.convertValues(source["items"], JobItem);
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.finishedAt = this.convertValues(source["finishedAt"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Preview {
	    path: string;
	    width: number;
	    height: number;
	    format: string;
	    size: number;
	    dataUrl?: string;
	    truncated?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Preview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.format = source["format"];
	        this.size = source["size"];
	        this.dataUrl = source["dataUrl"];
	        this.truncated = source["truncated"];
	    }
	}
	export class PreviewOptions {
	    maxDimension?: number;
	    maxPixels?: number;

	    static createFrom(source: any = {}) {
	        return new PreviewOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxDimension = source["maxDimension"];
	        this.maxPixels = source["maxPixels"];
	    }
	}
	export class QRCodeDecodeResult {
	    path: string;
	    width: number;
	    height: number;
	    format: string;
	    size: number;
	    text: string;
	    textBytes: number;
	    truncated?: boolean;

	    static createFrom(source: any = {}) {
	        return new QRCodeDecodeResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.format = source["format"];
	        this.size = source["size"];
	        this.text = source["text"];
	        this.textBytes = source["textBytes"];
	        this.truncated = source["truncated"];
	    }
	}
	export class QRCodePreview {
	    dataUrl: string;
	    size: number;

	    static createFrom(source: any = {}) {
	        return new QRCodePreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dataUrl = source["dataUrl"];
	        this.size = source["size"];
	    }
	}
	export class WatermarkPreview {
	    path: string;
	    width: number;
	    height: number;
	    beforeDataUrl?: string;
	    afterDataUrl?: string;
	    truncated?: boolean;

	    static createFrom(source: any = {}) {
	        return new WatermarkPreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.beforeDataUrl = source["beforeDataUrl"];
	        this.afterDataUrl = source["afterDataUrl"];
	        this.truncated = source["truncated"];
	    }
	}

}

export namespace platform {
	
	export class UpdateAsset {
	    id: string;
	    version: string;
	    platform: string;
	    architecture: string;
	    url: string;
	    sha256: string;
	    signature: string;
	    size?: number;
	
	    static createFrom(source: any = {}) {
	        return new UpdateAsset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.version = source["version"];
	        this.platform = source["platform"];
	        this.architecture = source["architecture"];
	        this.url = source["url"];
	        this.sha256 = source["sha256"];
	        this.signature = source["signature"];
	        this.size = source["size"];
	    }
	}
	export class UpdateInfo {
	    available: boolean;
	    version: string;
	    url?: string;
	    notes?: string;
	    assetId?: string;
	    sha256?: string;
	    signature?: string;
	    assets?: UpdateAsset[];
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.version = source["version"];
	        this.url = source["url"];
	        this.notes = source["notes"];
	        this.assetId = source["assetId"];
	        this.sha256 = source["sha256"];
	        this.signature = source["signature"];
	        this.assets = this.convertValues(source["assets"], UpdateAsset);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace tools {
	
	export class WatermarkOptions {
	    text: string;
	    fontFamily?: string;
	    fontSize?: number;
	    color?: string;
	    opacity?: number;
	    position?: string;
	    margin?: number;
	    rotation?: number;
	    tile?: boolean;
	    spacing?: number;
	    shadow?: boolean;

	    static createFrom(source: any = {}) {
	        return new WatermarkOptions(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.fontFamily = source["fontFamily"];
	        this.fontSize = source["fontSize"];
	        this.color = source["color"];
	        this.opacity = source["opacity"];
	        this.position = source["position"];
	        this.margin = source["margin"];
	        this.rotation = source["rotation"];
	        this.tile = source["tile"];
	        this.spacing = source["spacing"];
	        this.shadow = source["shadow"];
	    }
	}
	export class JobRequest {
	    tool: string;
	    inputs: string[];
	    inputRelativeDirs?: Record<string, string>;
	    outputDirectory: string;
	    format?: string;
	    quality?: number;
	    targetBytes?: number;
	    lossless?: boolean;
	    preserveMetadata?: boolean;
	    recursive?: boolean;
	    dpi?: number;
	    pageRange?: string;
	    maxPixels?: number;
	    watermark?: WatermarkOptions;
	
	    static createFrom(source: any = {}) {
	        return new JobRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.inputs = source["inputs"];
	        this.inputRelativeDirs = source["inputRelativeDirs"];
	        this.outputDirectory = source["outputDirectory"];
	        this.format = source["format"];
	        this.quality = source["quality"];
	        this.targetBytes = source["targetBytes"];
	        this.lossless = source["lossless"];
	        this.preserveMetadata = source["preserveMetadata"];
	        this.recursive = source["recursive"];
	        this.dpi = source["dpi"];
	        this.pageRange = source["pageRange"];
	        this.maxPixels = source["maxPixels"];
	        this.watermark = this.convertValues(source["watermark"], WatermarkOptions);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class QRCodeOptions {
	    text: string;
	    size: number;
	    errorCorrection: string;
	    foreground: string;
	    background: string;

	    static createFrom(source: any = {}) {
	        return new QRCodeOptions(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.size = source["size"];
	        this.errorCorrection = source["errorCorrection"];
	        this.foreground = source["foreground"];
	        this.background = source["background"];
	    }
	}
}
