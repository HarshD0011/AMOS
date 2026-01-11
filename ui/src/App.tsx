import { useEffect, useState, useMemo } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Landing } from './components/Landing';
import { ResourcePage } from './components/ResourcePage';
import { type Issue } from './components/IssueCard';

function App() {
    const [issues, setIssues] = useState<Issue[]>([]);
    const [loading, setLoading] = useState(true);

    const fetchIssues = async () => {
        try {
            const response = await fetch('/api/issues');
            if (!response.ok) throw new Error('Failed to fetch');
            const data = await response.json();
            data.sort((a: Issue, b: Issue) => {
                const statusPriority: Record<string, number> = { Failed: 0, Fixing: 1, Deploying: 2, Resolved: 3 };
                if (statusPriority[a.status] !== statusPriority[b.status]) {
                    return statusPriority[a.status] - statusPriority[b.status];
                }
                return new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime();
            });
            setIssues(data);
        } catch (err) {
            console.error(err);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchIssues();
        const interval = setInterval(fetchIssues, 2000);
        return () => clearInterval(interval);
    }, []);

    const groupedIssues = useMemo(() => {
        const groups: Record<string, Issue[]> = { Pod: [], Deployment: [], Job: [] };
        issues.forEach(issue => {
            if (groups[issue.kind]) {
                groups[issue.kind].push(issue);
            }
        });
        return groups;
    }, [issues]);

    return (
        <BrowserRouter>
            <Routes>
                <Route 
                    path="/" 
                    element={
                        <Landing 
                            podCount={groupedIssues.Pod.length}
                            deploymentCount={groupedIssues.Deployment.length}
                            jobCount={groupedIssues.Job.length}
                        />
                    } 
                />
                <Route 
                    path="/pods" 
                    element={
                        <ResourcePage 
                            title="Pods"
                            issues={groupedIssues.Pod}
                            loading={loading}
                            onRefresh={fetchIssues}
                            gradient="bg-gradient-to-r from-purple-600 to-purple-800"
                        />
                    } 
                />
                <Route 
                    path="/deployments" 
                    element={
                        <ResourcePage 
                            title="Deployments"
                            issues={groupedIssues.Deployment}
                            loading={loading}
                            onRefresh={fetchIssues}
                            gradient="bg-gradient-to-r from-indigo-600 to-indigo-800"
                        />
                    } 
                />
                <Route 
                    path="/jobs" 
                    element={
                        <ResourcePage 
                            title="Jobs"
                            issues={groupedIssues.Job}
                            loading={loading}
                            onRefresh={fetchIssues}
                            gradient="bg-gradient-to-r from-teal-600 to-teal-800"
                        />
                    } 
                />
            </Routes>
        </BrowserRouter>
    );
}

export default App;
