// import { grpc } from '@improbable-eng/grpc-web';
// For now, we'll implement without the actual grpc-web import to avoid type issues
// This will be replaced with proper gRPC-Web implementation later

// Types based on the proto file
export interface NodeInfo {
    name: string;
    owner_name: string;
    type: NodeType;
    modified?: Date;
}

export enum NodeType {
    UNKNOWN = 0,
    ROOM = 1,
    DIRECTORY = 2
}

export interface Status {
    ok: boolean;
    message: string;
}

export interface ListNodesRequest {
    path: string;
}

export interface ListNodesResponse {
    entries: NodeInfo[];
}

export interface CreateRoomRequest {
    path: string;
    owner_token: string;
}

export interface CreateRoomResponse {
    status: Status;
}

export interface CreateDirectoryRequest {
    path: string;
    owner_token: string;
}

export interface CreateDirectoryResponse {
    status: Status;
}

// gRPC-Web client configuration
const GRPC_WEB_ENDPOINT = 'http://localhost:8080'; // This will need to be configured

export class ChatshGrpcClient {
    private endpoint: string;

    constructor(endpoint: string = GRPC_WEB_ENDPOINT) {
        this.endpoint = endpoint;
    }

    // Mock implementation for now - will be replaced with actual gRPC-Web calls
    async listNodes(request: ListNodesRequest): Promise<ListNodesResponse> {
        // Mock data for testing
        const mockEntries: NodeInfo[] = [
            {
                name: 'general',
                owner_name: 'admin',
                type: NodeType.ROOM,
                modified: new Date()
            },
            {
                name: 'dev',
                owner_name: 'admin',
                type: NodeType.ROOM,
                modified: new Date()
            },
            {
                name: 'projects',
                owner_name: 'admin',
                type: NodeType.DIRECTORY,
                modified: new Date()
            },
            {
                name: 'random',
                owner_name: 'user',
                type: NodeType.ROOM,
                modified: new Date()
            }
        ];

        // Simulate network delay
        await new Promise(resolve => setTimeout(resolve, 100));

        return {
            entries: mockEntries
        };
    }

    async createRoom(request: CreateRoomRequest): Promise<CreateRoomResponse> {
        // Mock implementation
        await new Promise(resolve => setTimeout(resolve, 200));

        return {
            status: {
                ok: true,
                message: `Room created: ${request.path}`
            }
        };
    }

    async createDirectory(request: CreateDirectoryRequest): Promise<CreateDirectoryResponse> {
        // Mock implementation
        await new Promise(resolve => setTimeout(resolve, 200));

        return {
            status: {
                ok: true,
                message: `Directory created: ${request.path}`
            }
        };
    }

    async checkDirectoryExists(path: string): Promise<boolean> {
        // Mock implementation
        await new Promise(resolve => setTimeout(resolve, 50));

        // Mock: assume some paths exist
        const existingPaths = ['/home', '/home/projects', '/'];
        return existingPaths.includes(path);
    }
}

// Export a singleton instance
export const grpcClient = new ChatshGrpcClient();
