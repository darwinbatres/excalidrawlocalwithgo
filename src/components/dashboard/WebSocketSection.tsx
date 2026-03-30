import { StatCard } from "@/components/ui/StatCard";
import { SectionHeader } from "@/components/ui/SectionHeader";
import { Icons } from "@/components/ui/Icons";
import type { HubStats } from "@/types";

export function WebSocketSection({ hubStats }: { hubStats: HubStats | null }) {
  if (!hubStats) return null;
  return (
    <section>
      <SectionHeader title="WebSocket" description="Real-time collaboration hub" />
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        <StatCard
          title="Active Rooms"
          value={hubStats.activeRooms}
          subtitle="Boards with open WS connections"
          color="blue"
          icon={Icons.boards}
        />
        <StatCard
          title="Connected Clients"
          value={hubStats.totalClients}
          subtitle="WebSocket sessions"
          color="green"
          icon={Icons.users}
        />
        <StatCard
          title="Avg Clients/Room"
          value={hubStats.activeRooms > 0 ? (hubStats.totalClients / hubStats.activeRooms).toFixed(1) : "0"}
          subtitle="Collaborators per board"
          color="purple"
          icon={Icons.share}
        />
        <StatCard
          title="Messages In"
          value={hubStats.messagesIn.toLocaleString()}
          subtitle="Client → Server (since startup)"
          color="cyan"
          icon={Icons.connection}
        />
        <StatCard
          title="Messages Out"
          value={hubStats.messagesOut.toLocaleString()}
          subtitle="Server → Clients (since startup)"
          color="orange"
          icon={Icons.connection}
        />
        <StatCard
          title="Fan-out Ratio"
          value={hubStats.messagesIn > 0 ? (hubStats.messagesOut / hubStats.messagesIn).toFixed(1) + "x" : "0x"}
          subtitle="Lifetime avg outbound per inbound"
          color="indigo"
          icon={Icons.share}
        />
      </div>
    </section>
  );
}
