import React from 'react';
import { AlertCircle, CheckCircle, Loader2, Play, Terminal, FileText, ChevronDown, ChevronUp, Zap } from 'lucide-react';

export type IssueStatus = "Failed" | "Fixing" | "Deploying" | "Resolved";

export interface Issue {
    id: string;
    resource: string;
    namespace: string;
    kind: string;
    name: string;
    status: IssueStatus;
    errorMessage: string;
    logs: string;
    events: string;
    diagnosis: string;
    suggestion: string;
    updatedAt: string;
}

interface IssueCardProps {
    issue: Issue;
}

const statusConfig: Record<IssueStatus, { color: string; bgColor: string; icon: React.ReactNode; label: string }> = {
    Failed: { color: "text-red-400", bgColor: "bg-red-500", icon: <AlertCircle className="w-5 h-5 text-white" />, label: "Failed" },
    Fixing: { color: "text-yellow-400", bgColor: "bg-yellow-500", icon: <Loader2 className="w-5 h-5 text-white animate-spin" />, label: "Diagnosing" },
    Deploying: { color: "text-blue-400", bgColor: "bg-blue-500", icon: <Play className="w-5 h-5 text-white" />, label: "Fixing" },
    Resolved: { color: "text-green-400", bgColor: "bg-green-500", icon: <CheckCircle className="w-5 h-5 text-white" />, label: "Resolved" },
};

// Helper to extract recommendation and analysis from diagnosis
const parseDiagnosis = (diagnosis: string) => {
    const recMatch = diagnosis.match(/## RECOMMENDED SOLUTION\s*\n([\s\S]*?)(?=\n## ANALYSIS|$)/i);
    const analysisMatch = diagnosis.match(/## ANALYSIS\s*\n([\s\S]*?)$/i);
    
    return {
        recommendation: recMatch ? recMatch[1].trim() : diagnosis.slice(0, 300) + (diagnosis.length > 300 ? '...' : ''),
        fullTrace: analysisMatch ? analysisMatch[1].trim() : diagnosis,
    };
};

export const IssueCard: React.FC<IssueCardProps> = ({ issue }) => {
    const config = statusConfig[issue.status] || statusConfig.Failed;
    const [expanded, setExpanded] = React.useState(false);
    const [showTrace, setShowTrace] = React.useState(false);
    
    const { recommendation, fullTrace } = parseDiagnosis(issue.diagnosis || '');

    return (
        <div className={`bg-gray-800 rounded-xl shadow-lg overflow-hidden border border-gray-700 transition-all hover:border-gray-600 ${issue.status === 'Resolved' ? 'opacity-50' : ''}`}>
            {/* Header */}
            <div 
                className="p-4 flex items-center justify-between cursor-pointer hover:bg-gray-750 transition-colors"
                onClick={() => setExpanded(!expanded)}
            >
                <div className="flex items-center space-x-4">
                    <div className={`p-2.5 rounded-full ${config.bgColor} shadow-lg`}>
                        {config.icon}
                    </div>
                    <div>
                        <h3 className="font-bold text-lg text-white">{issue.name}</h3>
                        <p className="text-sm text-gray-400">{issue.namespace} • <span className={`font-medium ${config.color}`}>{config.label}</span></p>
                    </div>
                </div>
                <div className="flex items-center space-x-3">
                    <span className="text-xs text-gray-500">{new Date(issue.updatedAt).toLocaleTimeString()}</span>
                    {expanded ? <ChevronUp className="w-5 h-5 text-gray-500" /> : <ChevronDown className="w-5 h-5 text-gray-500" />}
                </div>
            </div>

            {/* Expanded Content */}
            {expanded && (
                <div className="bg-gray-850 p-5 border-t border-gray-700 space-y-4">
                    {/* Only show Error, Recommended Solution, and Analysis when NOT Resolved */}
                    {issue.status !== 'Resolved' && (
                        <>
                            {/* Error Message */}
                            <div>
                                <h4 className="font-semibold text-red-400 mb-2 flex items-center">
                                    <AlertCircle className="w-4 h-4 mr-2" /> Error
                                </h4>
                                <div className="bg-red-900/30 p-3 rounded-lg text-sm font-mono text-red-300 break-words border border-red-800/50">
                                    {issue.errorMessage}
                                </div>
                            </div>

                            {/* Recommended Solution */}
                            {recommendation && (
                                <div>
                                    <h4 className="font-semibold text-emerald-400 mb-2 flex items-center">
                                        <Zap className="w-4 h-4 mr-2" /> Recommended Solution
                                    </h4>
                                    <div className="bg-emerald-900/30 p-4 rounded-lg text-sm text-emerald-200 whitespace-pre-wrap border border-emerald-800/50 leading-relaxed">
                                        {recommendation}
                                    </div>
                                </div>
                            )}

                            {/* Full Model Trace (Collapsible) */}
                            {fullTrace && (
                                <div>
                                    <button 
                                        onClick={() => setShowTrace(!showTrace)}
                                        className="text-sm text-blue-400 hover:text-blue-300 font-medium flex items-center"
                                    >
                                        <FileText className="w-4 h-4 mr-1.5" />
                                        {showTrace ? 'Hide' : 'Show'} Full Analysis
                                    </button>
                                    {showTrace && (
                                        <div className="mt-2 bg-blue-900/20 p-4 rounded-lg text-sm text-blue-200 whitespace-pre-wrap border border-blue-800/50 leading-relaxed max-h-96 overflow-y-auto">
                                            {fullTrace}
                                        </div>
                                    )}
                                </div>
                            )}
                        </>
                    )}

                    {/* Events - Always shown */}
                    {issue.events && (
                        <details className="group" open={issue.status === 'Resolved'}>
                            <summary className="cursor-pointer text-gray-400 hover:text-gray-300 text-sm flex items-center font-medium">
                                <Terminal className="w-4 h-4 mr-2" />
                                View Events (kubectl describe)
                            </summary>
                            <div className="mt-2 text-xs bg-gray-900 text-amber-300 p-4 rounded-lg overflow-x-auto font-mono whitespace-pre max-h-64 overflow-y-auto border border-gray-700">
                                {issue.events}
                            </div>
                        </details>
                    )}

                    {/* Logs - Always shown with live indicator */}
                    <details className="group" open={issue.status === 'Resolved'}>
                        <summary className="cursor-pointer text-gray-400 hover:text-gray-300 text-sm flex items-center font-medium">
                            <Terminal className="w-4 h-4 mr-2" />
                            View Logs
                            <span className="ml-2 flex items-center text-green-400 text-xs">
                                <span className="relative flex h-2 w-2 mr-1">
                                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
                                    <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
                                </span>
                                Live
                            </span>
                        </summary>
                        <div className="mt-2 text-xs bg-gray-950 text-green-400 p-4 rounded-lg overflow-x-auto font-mono whitespace-pre max-h-64 overflow-y-auto border border-gray-700">
                            {issue.logs || "No logs available."}
                        </div>
                    </details>
                </div>
            )}
        </div>
    );
};
