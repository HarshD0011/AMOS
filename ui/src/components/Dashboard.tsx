import React, { useEffect, useState, useMemo } from 'react';
import { type Issue, IssueCard } from './IssueCard';
import { RefreshCw, Box, Layers, BriefcaseBusiness } from 'lucide-react';

type KindKey = 'Pod' | 'Deployment' | 'Job';

const kindConfig: Record<KindKey, { icon: React.ReactNode; label: string; color: string }> = {
    Pod: { icon: <Box className="w-5 h-5" />, label: "Pods", color: "text-purple-600" },
    Deployment: { icon: <Layers className="w-5 h-5" />, label: "Deployments", color: "text-indigo-600" },
    Job: { icon: <BriefcaseBusiness className="w-5 h-5" />, label: "Jobs", color: "text-teal-600" },
};

export const Dashboard: React.FC = () => {
    const [issues, setIssues] = useState<Issue[]>([]);
    const [error, setError] = useState<string | null>(null);
    const [loading, setLoading] = useState(true);

    const fetchIssues = async () => {
        try {
            const response = await fetch('/api/issues');
            if (!response.ok) {
                throw new Error('Failed to fetch issues');
            }
            const data = await response.json();
            // Sort: Failed first, then Fixing, Deploying, Resolved. Then by time.
            data.sort((a: Issue, b: Issue) => {
                const statusPriority: Record<string, number> = { Failed: 0, Fixing: 1, Deploying: 2, Resolved: 3 };
                if (statusPriority[a.status] !== statusPriority[b.status]) {
                    return statusPriority[a.status] - statusPriority[b.status];
                }
                return new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime();
            });
            setIssues(data);
            setError(null);
        } catch (err) {
            setError((err as Error).message);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchIssues();
        const interval = setInterval(fetchIssues, 2000);
        return () => clearInterval(interval);
    }, []);

    // Group issues by kind
    const groupedIssues = useMemo(() => {
        const groups: Record<KindKey, Issue[]> = { Pod: [], Deployment: [], Job: [] };
        issues.forEach(issue => {
            const kind = issue.kind as KindKey;
            if (groups[kind]) {
                groups[kind].push(issue);
            }
        });
        return groups;
    }, [issues]);

    const renderSection = (kind: KindKey) => {
        const config = kindConfig[kind];
        const kindIssues = groupedIssues[kind];
        
        return (
            <section key={kind} className="mb-8">
                <h2 className={`text-xl font-bold mb-4 flex items-center ${config.color}`}>
                    {config.icon}
                    <span className="ml-2">{config.label}</span>
                    <span className="ml-2 text-sm font-normal text-gray-400">({kindIssues.length})</span>
                </h2>
                {kindIssues.length === 0 ? (
                    <div className="text-center py-8 bg-gray-50 rounded-lg border border-dashed border-gray-200">
                        <p className="text-gray-400 text-sm">No issues for {config.label.toLowerCase()}</p>
                    </div>
                ) : (
                    <div className="space-y-4">
                        {kindIssues.map((issue) => (
                            <IssueCard key={issue.id} issue={issue} />
                        ))}
                    </div>
                )}
            </section>
        );
    };

    return (
        <div className="container mx-auto px-4 py-8 max-w-5xl">
            <header className="flex items-center justify-between mb-8 pb-4 border-b border-gray-200">
                <div>
                    <h1 className="text-3xl font-bold text-gray-900 tracking-tight">AMOS Dashboard</h1>
                    <p className="text-gray-500 mt-1">Agentic Mesh Observability System</p>
                </div>
                <button 
                    onClick={fetchIssues} 
                    className="p-2.5 bg-gray-100 hover:bg-gray-200 rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500"
                    title="Refresh"
                >
                    <RefreshCw className={`w-5 h-5 text-gray-600 ${loading ? 'animate-spin' : ''}`} />
                </button>
            </header>

            {error && (
                <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg mb-6 flex items-center">
                    <span className="font-bold mr-2">Connection Error:</span> {error} - Is the backend running?
                </div>
            )}

            {issues.length === 0 && !loading && !error ? (
                <div className="text-center py-20 bg-gray-50 rounded-xl border-2 border-dashed border-gray-200">
                    <p className="text-gray-500 text-lg">✅ No active issues detected</p>
                    <p className="text-sm text-gray-400 mt-2">AMOS is monitoring your cluster...</p>
                </div>
            ) : (
                <div>
                    {renderSection('Pod')}
                    {renderSection('Deployment')}
                    {renderSection('Job')}
                </div>
            )}
        </div>
    );
};
