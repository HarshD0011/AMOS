import { useNavigate } from 'react-router-dom';
import { type Issue, IssueCard } from './IssueCard';
import { ArrowLeft, RefreshCw } from 'lucide-react';

interface ResourcePageProps {
    title: string;
    issues: Issue[];
    loading: boolean;
    onRefresh: () => void;
    gradient: string;
}

export const ResourcePage: React.FC<ResourcePageProps> = ({ title, issues, loading, onRefresh, gradient }) => {
    const navigate = useNavigate();

    return (
        <div className="min-h-screen bg-gray-900 text-white">
            {/* Header */}
            <header className={`${gradient} py-8 px-6 shadow-lg`}>
                <div className="max-w-5xl mx-auto flex items-center justify-between">
                    <div className="flex items-center space-x-4">
                        <button
                            onClick={() => navigate('/')}
                            className="p-2 bg-white/10 hover:bg-white/20 rounded-full transition-colors"
                        >
                            <ArrowLeft className="w-6 h-6" />
                        </button>
                        <div>
                            <h1 className="text-3xl font-bold">{title}</h1>
                            <p className="text-white/70 text-sm">{issues.length} issue(s) detected</p>
                        </div>
                    </div>
                    <button
                        onClick={onRefresh}
                        className="p-3 bg-white/10 hover:bg-white/20 rounded-full transition-colors"
                        title="Refresh"
                    >
                        <RefreshCw className={`w-5 h-5 ${loading ? 'animate-spin' : ''}`} />
                    </button>
                </div>
            </header>

            {/* Content */}
            <main className="max-w-5xl mx-auto px-6 py-8">
                {issues.length === 0 ? (
                    <div className="text-center py-20 bg-gray-800/50 rounded-xl border border-gray-700 border-dashed">
                        <p className="text-gray-400 text-lg">✅ No issues found</p>
                        <p className="text-gray-500 text-sm mt-2">This category is healthy</p>
                    </div>
                ) : (
                    <div className="space-y-4">
                        {issues.map((issue) => (
                            <IssueCard key={issue.id} issue={issue} />
                        ))}
                    </div>
                )}
            </main>
        </div>
    );
};
