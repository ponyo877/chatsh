import { createPromiseClient, PromiseClient } from "@bufbuild/connect";
import { createGrpcWebTransport } from "@bufbuild/connect-web";
import { ChatshService } from "./generated/chatsh_connect";
import {
    ListNodesRequest,
    // ListNodesResponse, // Response type is inferred from client call
    CreateRoomRequest,
    // CreateRoomResponse,
    CreateDirectoryRequest,
    // CreateDirectoryResponse,
    CheckDirectoryExistsRequest,
    // CheckDirectoryExistsResponse,
    WriteMessageRequest,
    // WriteMessageResponse,
    ListMessagesRequest,
    // ListMessagesResponse,
    NodeInfo as GrpcNodeInfo, // Keep alias for clarity in mapping
    Message as GrpcMessage,   // Keep alias for clarity in mapping
    NodeType as GrpcNodeType,
    Status as GrpcStatus,
} from "./generated/chatsh_pb";
import { Timestamp } from "@bufbuild/protobuf"; // Correct Timestamp import

// Re-exporting types for internal use, mapping from gRPC generated types
export enum NodeType {
    UNKNOWN = GrpcNodeType.UNKNOWN,
    ROOM = GrpcNodeType.ROOM,
    DIRECTORY = GrpcNodeType.DIRECTORY
}

// Our application's internal types - these will be what our UI consumes
export interface NodeInfo {
    name: string;
    ownerName: string;
    type: NodeType;
    modified?: Date;
}

export interface Status {
    ok: boolean;
    message: string;
}

export interface ListNodesParams {
    path: string;
}

export interface ListNodesResult {
    entries: NodeInfo[];
}

export interface CreateRoomParams {
    path: string;
    ownerToken: string;
}

export interface CreateRoomResult {
    status: Status;
}

export interface CreateDirectoryParams {
    path: string;
    ownerToken: string;
}

export interface CreateDirectoryResult {
    status: Status;
}

export interface WriteMessageParams {
    textContent: string;
    destinationPath: string;
    ownerToken: string;
}

export interface WriteMessageResult {
    status: Status;
}

export interface ListMessagesParams {
    roomPath: string;
    limit?: number;
}

export interface ChatMessage {
    textContent: string;
    ownerName: string;
    created: Date;
}

export interface ListMessagesResult {
    messages: ChatMessage[];
}

// gRPC-Web client configuration
const GRPC_WEB_ENDPOINT = 'http://localhost:8080'; // Envoy proxy endpoint

export class ChatshGrpcClient {
    private client: PromiseClient<typeof ChatshService>;

    constructor(endpoint: string = GRPC_WEB_ENDPOINT) {
        const transport = createGrpcWebTransport({
            baseUrl: endpoint,
        });
        this.client = createPromiseClient(ChatshService, transport);
    }

    private grpcNodeInfoToNodeInfo(entry: GrpcNodeInfo): NodeInfo {
        return {
            name: entry.name,
            ownerName: entry.ownerName,
            type: entry.type as unknown as NodeType, // Enum mapping
            modified: entry.modified ? entry.modified.toDate() : undefined,
        };
    }

    private grpcStatusToStatus(status?: GrpcStatus): Status {
        return {
            ok: status?.ok ?? false,
            message: status?.message ?? '',
        };
    }

    private grpcMessageToChatMessage(msg: GrpcMessage): ChatMessage {
        return {
            textContent: msg.textContent,
            ownerName: msg.ownerName,
            created: msg.created ? msg.created.toDate() : new Date(),
        };
    }

    async listNodes(params: ListNodesParams): Promise<ListNodesResult> {
        const request = new ListNodesRequest({ path: params.path });
        try {
            const response = await this.client.listNodes(request);
            const entries: NodeInfo[] = response.entries.map(this.grpcNodeInfoToNodeInfo);
            return { entries };
        } catch (error) {
            console.error('Connect listNodes error:', error);
            throw error;
        }
    }

    async createRoom(params: CreateRoomParams): Promise<CreateRoomResult> {
        const request = new CreateRoomRequest({ path: params.path, ownerToken: params.ownerToken });
        try {
            const response = await this.client.createRoom(request);
            return { status: this.grpcStatusToStatus(response.status) };
        } catch (error) {
            console.error('Connect createRoom error:', error);
            throw error;
        }
    }

    async createDirectory(params: CreateDirectoryParams): Promise<CreateDirectoryResult> {
        const request = new CreateDirectoryRequest({ path: params.path, ownerToken: params.ownerToken });
        try {
            const response = await this.client.createDirectory(request);
            return { status: this.grpcStatusToStatus(response.status) };
        } catch (error) {
            console.error('Connect createDirectory error:', error);
            throw error;
        }
    }

    async checkDirectoryExists(path: string): Promise<boolean> {
        const request = new CheckDirectoryExistsRequest({ path });
        try {
            const response = await this.client.checkDirectoryExists(request);
            return response.exists;
        } catch (error) {
            console.error('Connect checkDirectoryExists error:', error);
            throw error;
        }
    }

    async writeMessage(params: WriteMessageParams): Promise<WriteMessageResult> {
        const request = new WriteMessageRequest({
            textContent: params.textContent,
            destinationPath: params.destinationPath,
            ownerToken: params.ownerToken,
        });
        try {
            const response = await this.client.writeMessage(request);
            return { status: this.grpcStatusToStatus(response.status) };
        } catch (error) {
            console.error('Connect writeMessage error:', error);
            throw error;
        }
    }

    async listMessages(params: ListMessagesParams): Promise<ListMessagesResult> {
        const request = new ListMessagesRequest({
            roomPath: params.roomPath,
            limit: params.limit ?? 0,
        });
        try {
            const response = await this.client.listMessages(request);
            const messages: ChatMessage[] = response.messages.map(this.grpcMessageToChatMessage);
            return { messages };
        } catch (error) {
            console.error('Connect listMessages error:', error);
            throw error;
        }
    }
}

// Export a singleton instance
export const grpcClient = new ChatshGrpcClient();
