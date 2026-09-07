export namespace domain {

	export class PersistenceRolloutConfig {
	    configured?: boolean;
	    journalEnabled: boolean;
	    dualWriteValidation: boolean;
	    readPath?: string;

	    static createFrom(source: any = {}) {
	        return new PersistenceRolloutConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.configured = source["configured"];
	        this.journalEnabled = source["journalEnabled"];
	        this.dualWriteValidation = source["dualWriteValidation"];
	        this.readPath = source["readPath"];
	    }
	}
	export class ModelRef {
	    providerId: string;
	    modelId: string;

	    static createFrom(source: any = {}) {
	        return new ModelRef(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerId = source["providerId"];
	        this.modelId = source["modelId"];
	    }
	}
	export class ProviderSettings {
	    custom?: Record<string, ProviderConfig>;
	    disabled?: string[];

	    static createFrom(source: any = {}) {
	        return new ProviderSettings(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.custom = this.convertValues(source["custom"], ProviderConfig, true);
	        this.disabled = source["disabled"];
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
	export class ProviderConfig {
	    id: string;
	    type: string;
	    baseUrl?: string;
	    apiKey?: string;
	    apiKeyEnv?: string;
	    model?: string;
	    headers?: Record<string, string>;

	    static createFrom(source: any = {}) {
	        return new ProviderConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.baseUrl = source["baseUrl"];
	        this.apiKey = source["apiKey"];
	        this.apiKeyEnv = source["apiKeyEnv"];
	        this.model = source["model"];
	        this.headers = source["headers"];
	    }
	}
	export class AppConfig {
	    initialized: boolean;
	    provider?: ProviderConfig;
	    providers?: ProviderSettings;
	    defaultModel?: ModelRef;
	    reasoningEffort?: string;
	    serviceTier?: string;
	    persistence?: PersistenceRolloutConfig;
	    configPath?: string;

	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.initialized = source["initialized"];
	        this.provider = this.convertValues(source["provider"], ProviderConfig);
	        this.providers = this.convertValues(source["providers"], ProviderSettings);
	        this.defaultModel = this.convertValues(source["defaultModel"], ModelRef);
	        this.reasoningEffort = source["reasoningEffort"];
	        this.serviceTier = source["serviceTier"];
	        this.persistence = this.convertValues(source["persistence"], PersistenceRolloutConfig);
	        this.configPath = source["configPath"];
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
	export class AppendEventRequest {
	    sessionId: string;
	    turnId?: string;
	    type: string;
	    role?: string;
	    visibility?: string;
	    content?: string;
	    payload?: Record<string, any>;
	    tokenCount?: number;

	    static createFrom(source: any = {}) {
	        return new AppendEventRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.turnId = source["turnId"];
	        this.type = source["type"];
	        this.role = source["role"];
	        this.visibility = source["visibility"];
	        this.content = source["content"];
	        this.payload = source["payload"];
	        this.tokenCount = source["tokenCount"];
	    }
	}
	export class ApprovePermissionRequestInput {
	    requestId: string;
	    remember?: boolean;

	    static createFrom(source: any = {}) {
	        return new ApprovePermissionRequestInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requestId = source["requestId"];
	        this.remember = source["remember"];
	    }
	}
	export class AssistantProject {
	    id: string;
	    name: string;
	    rootPath: string;
	    gitBranch?: string;
	    gitDirty?: boolean;
	    gitAvailable: boolean;
	    timeOpened: string;
	    timeUpdated: string;

	    static createFrom(source: any = {}) {
	        return new AssistantProject(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.rootPath = source["rootPath"];
	        this.gitBranch = source["gitBranch"];
	        this.gitDirty = source["gitDirty"];
	        this.gitAvailable = source["gitAvailable"];
	        this.timeOpened = source["timeOpened"];
	        this.timeUpdated = source["timeUpdated"];
	    }
	}
	export class AuthInfo {
	    type: string;
	    connected: boolean;
	    source: string;
	    environment?: string;
	    lastValidatedAt?: string;
	    connectedAt?: string;

	    static createFrom(source: any = {}) {
	        return new AuthInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.connected = source["connected"];
	        this.source = source["source"];
	        this.environment = source["environment"];
	        this.lastValidatedAt = source["lastValidatedAt"];
	        this.connectedAt = source["connectedAt"];
	    }
	}
	export class BuildSessionContextRequest {
	    sessionId: string;
	    currentInput?: string;
	    maxTokens?: number;
	    characterBudget?: number;

	    static createFrom(source: any = {}) {
	        return new BuildSessionContextRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.currentInput = source["currentInput"];
	        this.maxTokens = source["maxTokens"];
	        this.characterBudget = source["characterBudget"];
	    }
	}
	export class ContextSection {
	    name: string;
	    content: string;
	    truncated?: boolean;

	    static createFrom(source: any = {}) {
	        return new ContextSection(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.content = source["content"];
	        this.truncated = source["truncated"];
	    }
	}
	export class BuildSessionContextResult {
	    sessionId: string;
	    sections: ContextSection[];
	    estimatedTokens: number;
	    characterBudget?: number;
	    truncatedSections?: string[];

	    static createFrom(source: any = {}) {
	        return new BuildSessionContextResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.sections = this.convertValues(source["sections"], ContextSection);
	        this.estimatedTokens = source["estimatedTokens"];
	        this.characterBudget = source["characterBudget"];
	        this.truncatedSections = source["truncatedSections"];
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
	export class CancelTurnRequest {
	    turnId: string;
	    reason?: string;

	    static createFrom(source: any = {}) {
	        return new CancelTurnRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.turnId = source["turnId"];
	        this.reason = source["reason"];
	    }
	}
	export class ProviderAccountInfo {
	    id: string;
	    providerId: string;
	    method: string;
	    accountId: string;
	    displayName: string;
	    connectedAt?: string;

	    static createFrom(source: any = {}) {
	        return new ProviderAccountInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.providerId = source["providerId"];
	        this.method = source["method"];
	        this.accountId = source["accountId"];
	        this.displayName = source["displayName"];
	        this.connectedAt = source["connectedAt"];
	    }
	}
	export class ProviderAuthMethod {
	    id: string;
	    label: string;
	    stable: boolean;
	    available: boolean;
	    description?: string;

	    static createFrom(source: any = {}) {
	        return new ProviderAuthMethod(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.stable = source["stable"];
	        this.available = source["available"];
	        this.description = source["description"];
	    }
	}
	export class ModelInfo {
	    id: string;
	    providerId: string;
	    name: string;
	    recommended?: boolean;
	    capabilities?: string[];

	    static createFrom(source: any = {}) {
	        return new ModelInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.providerId = source["providerId"];
	        this.name = source["name"];
	        this.recommended = source["recommended"];
	        this.capabilities = source["capabilities"];
	    }
	}
	export class ProviderInfo {
	    id: string;
	    name: string;
	    type: string;
	    baseUrl?: string;
	    builtIn: boolean;
	    custom: boolean;
	    connected: boolean;
	    connectionSource?: string;
	    environment?: string;
	    defaultModelId?: string;
	    models: ModelInfo[];
	    authMethods: ProviderAuthMethod[];
	    auth?: AuthInfo;
	    accounts?: ProviderAccountInfo[];

	    static createFrom(source: any = {}) {
	        return new ProviderInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.baseUrl = source["baseUrl"];
	        this.builtIn = source["builtIn"];
	        this.custom = source["custom"];
	        this.connected = source["connected"];
	        this.connectionSource = source["connectionSource"];
	        this.environment = source["environment"];
	        this.defaultModelId = source["defaultModelId"];
	        this.models = this.convertValues(source["models"], ModelInfo);
	        this.authMethods = this.convertValues(source["authMethods"], ProviderAuthMethod);
	        this.auth = this.convertValues(source["auth"], AuthInfo);
	        this.accounts = this.convertValues(source["accounts"], ProviderAccountInfo);
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
	export class CatalogState {
	    providers: ProviderInfo[];
	    models: ModelInfo[];
	    connected: string[];
	    defaultModel?: ModelRef;
	    connectedProviders?: ProviderInfo[];
	    popularProviders?: ProviderInfo[];
	    customProviders?: ProviderInfo[];

	    static createFrom(source: any = {}) {
	        return new CatalogState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providers = this.convertValues(source["providers"], ProviderInfo);
	        this.models = this.convertValues(source["models"], ModelInfo);
	        this.connected = source["connected"];
	        this.defaultModel = this.convertValues(source["defaultModel"], ModelRef);
	        this.connectedProviders = this.convertValues(source["connectedProviders"], ProviderInfo);
	        this.popularProviders = this.convertValues(source["popularProviders"], ProviderInfo);
	        this.customProviders = this.convertValues(source["customProviders"], ProviderInfo);
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
	export class CodingContext {
	    id: string;
	    sessionId: string;
	    projectPath: string;
	    gitBranch?: string;
	    commitSha?: string;
	    repoUrl?: string;
	    changedFiles?: string[];
	    languageStack?: string[];
	    packageManager?: string;
	    cwd?: string;
	    permissions?: string[];
	    lastCommand?: string;
	    timeCreated: string;
	    timeUpdated: string;

	    static createFrom(source: any = {}) {
	        return new CodingContext(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionId = source["sessionId"];
	        this.projectPath = source["projectPath"];
	        this.gitBranch = source["gitBranch"];
	        this.commitSha = source["commitSha"];
	        this.repoUrl = source["repoUrl"];
	        this.changedFiles = source["changedFiles"];
	        this.languageStack = source["languageStack"];
	        this.packageManager = source["packageManager"];
	        this.cwd = source["cwd"];
	        this.permissions = source["permissions"];
	        this.lastCommand = source["lastCommand"];
	        this.timeCreated = source["timeCreated"];
	        this.timeUpdated = source["timeUpdated"];
	    }
	}
	export class CompleteTurnRequest {
	    turnId: string;

	    static createFrom(source: any = {}) {
	        return new CompleteTurnRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.turnId = source["turnId"];
	    }
	}

	export class CreateCheckpointRequest {
	    sessionId: string;
	    diffSummary?: string;
	    conversationSummary?: string;
	    openTodos?: string[];
	    knownIssues?: string[];
	    nextSuggestedAction?: string;

	    static createFrom(source: any = {}) {
	        return new CreateCheckpointRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.diffSummary = source["diffSummary"];
	        this.conversationSummary = source["conversationSummary"];
	        this.openTodos = source["openTodos"];
	        this.knownIssues = source["knownIssues"];
	        this.nextSuggestedAction = source["nextSuggestedAction"];
	    }
	}
	export class CreateSessionRequest {
	    type?: string;
	    source?: string;
	    title?: string;
	    goal?: string;
	    projectPath?: string;
	    model?: ModelRef;
	    modelSnapshot?: string;
	    systemPromptSnapshot?: string;
	    metadata?: Record<string, string>;

	    static createFrom(source: any = {}) {
	        return new CreateSessionRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.source = source["source"];
	        this.title = source["title"];
	        this.goal = source["goal"];
	        this.projectPath = source["projectPath"];
	        this.model = this.convertValues(source["model"], ModelRef);
	        this.modelSnapshot = source["modelSnapshot"];
	        this.systemPromptSnapshot = source["systemPromptSnapshot"];
	        this.metadata = source["metadata"];
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
	export class CreateSummaryRequest {
	    sessionId: string;
	    fromEventId?: string;
	    toEventId?: string;
	    summary?: string;
	    facts?: string[];
	    decisions?: string[];
	    openTasks?: string[];
	    changedFiles?: string[];
	    nextSuggestedAction?: string;

	    static createFrom(source: any = {}) {
	        return new CreateSummaryRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.fromEventId = source["fromEventId"];
	        this.toEventId = source["toEventId"];
	        this.summary = source["summary"];
	        this.facts = source["facts"];
	        this.decisions = source["decisions"];
	        this.openTasks = source["openTasks"];
	        this.changedFiles = source["changedFiles"];
	        this.nextSuggestedAction = source["nextSuggestedAction"];
	    }
	}
	export class CreateToolCallRequest {
	    id?: string;
	    sessionId: string;
	    turnId?: string;
	    eventId?: string;
	    name: string;
	    arguments?: Record<string, any>;
	    status?: string;
	    resultSummary?: string;
	    result?: Record<string, any>;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new CreateToolCallRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionId = source["sessionId"];
	        this.turnId = source["turnId"];
	        this.eventId = source["eventId"];
	        this.name = source["name"];
	        this.arguments = source["arguments"];
	        this.status = source["status"];
	        this.resultSummary = source["resultSummary"];
	        this.result = source["result"];
	        this.error = source["error"];
	    }
	}
	export class DenyPermissionRequestInput {
	    requestId: string;
	    remember?: boolean;
	    reason?: string;

	    static createFrom(source: any = {}) {
	        return new DenyPermissionRequestInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requestId = source["requestId"];
	        this.remember = source["remember"];
	        this.reason = source["reason"];
	    }
	}
	export class FailTurnRequest {
	    turnId: string;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new FailTurnRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.turnId = source["turnId"];
	        this.error = source["error"];
	    }
	}
	export class ForkSessionRequest {
	    sessionId: string;
	    title?: string;
	    goal?: string;

	    static createFrom(source: any = {}) {
	        return new ForkSessionRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.title = source["title"];
	        this.goal = source["goal"];
	    }
	}
	export class ListSessionsRequest {
	    type?: string;
	    status?: string;
	    source?: string;
	    projectPath?: string;
	    search?: string;
	    includeDeleted?: boolean;
	    limit?: number;

	    static createFrom(source: any = {}) {
	        return new ListSessionsRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.status = source["status"];
	        this.source = source["source"];
	        this.projectPath = source["projectPath"];
	        this.search = source["search"];
	        this.includeDeleted = source["includeDeleted"];
	        this.limit = source["limit"];
	    }
	}

	export class ModelPreferencesInput {
	    model?: ModelRef;
	    reasoningEffort?: string;
	    serviceTier?: string;

	    static createFrom(source: any = {}) {
	        return new ModelPreferencesInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = this.convertValues(source["model"], ModelRef);
	        this.reasoningEffort = source["reasoningEffort"];
	        this.serviceTier = source["serviceTier"];
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

	export class PermissionRequest {
	    id: string;
	    sessionId?: string;
	    turnId?: string;
	    toolCallId?: string;
	    toolName: string;
	    action: string;
	    paths?: string[];
	    arguments?: Record<string, any>;
	    status: string;
	    remember?: boolean;
	    reason?: string;
	    timeCreated: string;
	    timeUpdated: string;

	    static createFrom(source: any = {}) {
	        return new PermissionRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionId = source["sessionId"];
	        this.turnId = source["turnId"];
	        this.toolCallId = source["toolCallId"];
	        this.toolName = source["toolName"];
	        this.action = source["action"];
	        this.paths = source["paths"];
	        this.arguments = source["arguments"];
	        this.status = source["status"];
	        this.remember = source["remember"];
	        this.reason = source["reason"];
	        this.timeCreated = source["timeCreated"];
	        this.timeUpdated = source["timeUpdated"];
	    }
	}

	export class SessionEvent {
	    id: string;
	    sessionId: string;
	    turnId?: string;
	    type: string;
	    role?: string;
	    visibility: string;
	    content?: string;
	    payload?: Record<string, any>;
	    tokenCount?: number;
	    timeCreated: string;

	    static createFrom(source: any = {}) {
	        return new SessionEvent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionId = source["sessionId"];
	        this.turnId = source["turnId"];
	        this.type = source["type"];
	        this.role = source["role"];
	        this.visibility = source["visibility"];
	        this.content = source["content"];
	        this.payload = source["payload"];
	        this.tokenCount = source["tokenCount"];
	        this.timeCreated = source["timeCreated"];
	    }
	}
	export class Turn {
	    id: string;
	    sessionId: string;
	    status: string;
	    userEventId?: string;
	    error?: string;
	    timeCreated: string;
	    timeCompleted?: string;
	    timeUpdated: string;

	    static createFrom(source: any = {}) {
	        return new Turn(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionId = source["sessionId"];
	        this.status = source["status"];
	        this.userEventId = source["userEventId"];
	        this.error = source["error"];
	        this.timeCreated = source["timeCreated"];
	        this.timeCompleted = source["timeCompleted"];
	        this.timeUpdated = source["timeUpdated"];
	    }
	}
	export class PreparedSessionTurn {
	    turn: Turn;
	    model?: ModelRef;
	    userEvent: SessionEvent;
	    assistantEvent?: SessionEvent;

	    static createFrom(source: any = {}) {
	        return new PreparedSessionTurn(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.turn = this.convertValues(source["turn"], Turn);
	        this.model = this.convertValues(source["model"], ModelRef);
	        this.userEvent = this.convertValues(source["userEvent"], SessionEvent);
	        this.assistantEvent = this.convertValues(source["assistantEvent"], SessionEvent);
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


	export class ProviderAuthStartInput {
	    providerId: string;
	    method: string;

	    static createFrom(source: any = {}) {
	        return new ProviderAuthStartInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerId = source["providerId"];
	        this.method = source["method"];
	    }
	}
	export class ProviderAuthStartResult {
	    providerId: string;
	    method: string;
	    status: string;
	    url?: string;
	    instructions?: string;
	    userCode?: string;
	    expiresAt?: string;
	    authorization?: string;

	    static createFrom(source: any = {}) {
	        return new ProviderAuthStartResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerId = source["providerId"];
	        this.method = source["method"];
	        this.status = source["status"];
	        this.url = source["url"];
	        this.instructions = source["instructions"];
	        this.userCode = source["userCode"];
	        this.expiresAt = source["expiresAt"];
	        this.authorization = source["authorization"];
	    }
	}
	export class ProviderAuthStatus {
	    providerId: string;
	    method: string;
	    status: string;
	    error?: string;
	    accountId?: string;
	    instructions?: string;
	    userCode?: string;

	    static createFrom(source: any = {}) {
	        return new ProviderAuthStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerId = source["providerId"];
	        this.method = source["method"];
	        this.status = source["status"];
	        this.error = source["error"];
	        this.accountId = source["accountId"];
	        this.instructions = source["instructions"];
	        this.userCode = source["userCode"];
	    }
	}

	export class ProviderConnectInput {
	    providerId: string;
	    name?: string;
	    type?: string;
	    baseUrl?: string;
	    apiKey?: string;
	    apiKeyEnv?: string;
	    modelId?: string;
	    method?: string;
	    headers?: Record<string, string>;

	    static createFrom(source: any = {}) {
	        return new ProviderConnectInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providerId = source["providerId"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.baseUrl = source["baseUrl"];
	        this.apiKey = source["apiKey"];
	        this.apiKeyEnv = source["apiKeyEnv"];
	        this.modelId = source["modelId"];
	        this.method = source["method"];
	        this.headers = source["headers"];
	    }
	}


	export class SessionCheckpoint {
	    id: string;
	    sessionId: string;
	    branch?: string;
	    commitSha?: string;
	    changedFiles?: string[];
	    diffSummary?: string;
	    conversationSummary?: string;
	    openTodos?: string[];
	    knownIssues?: string[];
	    nextSuggestedAction?: string;
	    timeCreated: string;

	    static createFrom(source: any = {}) {
	        return new SessionCheckpoint(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionId = source["sessionId"];
	        this.branch = source["branch"];
	        this.commitSha = source["commitSha"];
	        this.changedFiles = source["changedFiles"];
	        this.diffSummary = source["diffSummary"];
	        this.conversationSummary = source["conversationSummary"];
	        this.openTodos = source["openTodos"];
	        this.knownIssues = source["knownIssues"];
	        this.nextSuggestedAction = source["nextSuggestedAction"];
	        this.timeCreated = source["timeCreated"];
	    }
	}
	export class SessionSummary {
	    id: string;
	    sessionId: string;
	    fromEventId?: string;
	    toEventId?: string;
	    summary: string;
	    facts?: string[];
	    decisions?: string[];
	    openTasks?: string[];
	    changedFiles?: string[];
	    nextSuggestedAction?: string;
	    timeCreated: string;

	    static createFrom(source: any = {}) {
	        return new SessionSummary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionId = source["sessionId"];
	        this.fromEventId = source["fromEventId"];
	        this.toEventId = source["toEventId"];
	        this.summary = source["summary"];
	        this.facts = source["facts"];
	        this.decisions = source["decisions"];
	        this.openTasks = source["openTasks"];
	        this.changedFiles = source["changedFiles"];
	        this.nextSuggestedAction = source["nextSuggestedAction"];
	        this.timeCreated = source["timeCreated"];
	    }
	}
	export class ResumeRecap {
	    sessionId: string;
	    title: string;
	    goal?: string;
	    latestSummary?: SessionSummary;
	    projectPath?: string;
	    branch?: string;
	    changedFiles?: string[];
	    openTodos?: string[];
	    lastCommand?: string;
	    nextSuggestedAction?: string;
	    updatedTime: string;
	    latestCheckpoint?: SessionCheckpoint;
	    recentEvents?: SessionEvent[];

	    static createFrom(source: any = {}) {
	        return new ResumeRecap(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.title = source["title"];
	        this.goal = source["goal"];
	        this.latestSummary = this.convertValues(source["latestSummary"], SessionSummary);
	        this.projectPath = source["projectPath"];
	        this.branch = source["branch"];
	        this.changedFiles = source["changedFiles"];
	        this.openTodos = source["openTodos"];
	        this.lastCommand = source["lastCommand"];
	        this.nextSuggestedAction = source["nextSuggestedAction"];
	        this.updatedTime = source["updatedTime"];
	        this.latestCheckpoint = this.convertValues(source["latestCheckpoint"], SessionCheckpoint);
	        this.recentEvents = this.convertValues(source["recentEvents"], SessionEvent);
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
	export class ResumeSessionRequest {
	    sessionId?: string;
	    projectPath?: string;

	    static createFrom(source: any = {}) {
	        return new ResumeSessionRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.projectPath = source["projectPath"];
	    }
	}
	export class Session {
	    id: string;
	    type: string;
	    status: string;
	    source: string;
	    title: string;
	    goal?: string;
	    summary?: string;
	    parentSessionId?: string;
	    forkedFromSessionId?: string;
	    projectPath?: string;
	    model?: ModelRef;
	    modelSnapshot?: string;
	    systemPromptSnapshot?: string;
	    tokenCount?: number;
	    costMicros?: number;
	    metadata?: Record<string, string>;
	    archivedAt?: string;
	    deletedAt?: string;
	    timeCreated: string;
	    timeUpdated: string;

	    static createFrom(source: any = {}) {
	        return new Session(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.source = source["source"];
	        this.title = source["title"];
	        this.goal = source["goal"];
	        this.summary = source["summary"];
	        this.parentSessionId = source["parentSessionId"];
	        this.forkedFromSessionId = source["forkedFromSessionId"];
	        this.projectPath = source["projectPath"];
	        this.model = this.convertValues(source["model"], ModelRef);
	        this.modelSnapshot = source["modelSnapshot"];
	        this.systemPromptSnapshot = source["systemPromptSnapshot"];
	        this.tokenCount = source["tokenCount"];
	        this.costMicros = source["costMicros"];
	        this.metadata = source["metadata"];
	        this.archivedAt = source["archivedAt"];
	        this.deletedAt = source["deletedAt"];
	        this.timeCreated = source["timeCreated"];
	        this.timeUpdated = source["timeUpdated"];
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



	export class StartTurnRequest {
	    sessionId: string;
	    userEventId?: string;

	    static createFrom(source: any = {}) {
	        return new StartTurnRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.userEventId = source["userEventId"];
	    }
	}
	export class SubmitSessionMessageRequest {
	    sessionId: string;
	    text: string;
	    model?: ModelRef;
	    reasoningEffort?: string;
	    serviceTier?: string;

	    static createFrom(source: any = {}) {
	        return new SubmitSessionMessageRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.text = source["text"];
	        this.model = this.convertValues(source["model"], ModelRef);
	        this.reasoningEffort = source["reasoningEffort"];
	        this.serviceTier = source["serviceTier"];
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
	export class ToolCall {
	    id: string;
	    sessionId: string;
	    turnId?: string;
	    eventId?: string;
	    name: string;
	    arguments?: Record<string, any>;
	    status: string;
	    resultSummary?: string;
	    result?: Record<string, any>;
	    error?: string;
	    timeCreated: string;
	    timeUpdated: string;

	    static createFrom(source: any = {}) {
	        return new ToolCall(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionId = source["sessionId"];
	        this.turnId = source["turnId"];
	        this.eventId = source["eventId"];
	        this.name = source["name"];
	        this.arguments = source["arguments"];
	        this.status = source["status"];
	        this.resultSummary = source["resultSummary"];
	        this.result = source["result"];
	        this.error = source["error"];
	        this.timeCreated = source["timeCreated"];
	        this.timeUpdated = source["timeUpdated"];
	    }
	}

	export class UpdateSessionRequest {
	    sessionId: string;
	    title?: string;
	    goal?: string;
	    summary?: string;
	    status?: string;

	    static createFrom(source: any = {}) {
	        return new UpdateSessionRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.title = source["title"];
	        this.goal = source["goal"];
	        this.summary = source["summary"];
	        this.status = source["status"];
	    }
	}

}
