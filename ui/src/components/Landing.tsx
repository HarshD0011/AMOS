import { useNavigate } from 'react-router-dom';
import { Box, Layers, BriefcaseBusiness } from 'lucide-react';

interface CategoryCardProps {
    title: string;
    icon: React.ReactNode;
    count: number;
    color: string;
    gradient: string;
    onClick: () => void;
}

const CategoryCard: React.FC<CategoryCardProps> = ({ title, icon, count, gradient, onClick }) => {
    return (
        <div
            onClick={onClick}
            className={`
                cursor-pointer rounded-2xl p-8 
                ${gradient}
                transform transition-all duration-300 ease-out
                hover:scale-105 hover:shadow-2xl hover:shadow-purple-500/20
                active:scale-95
                border border-white/10
                group
            `}
        >
            <div className="flex flex-col items-center text-center space-y-4">
                <div className="p-4 bg-white/10 rounded-2xl group-hover:bg-white/20 transition-colors duration-300">
                    {icon}
                </div>
                <h2 className="text-2xl font-bold text-white tracking-tight">{title}</h2>
                <div className="flex items-center space-x-2">
                    <span className="text-5xl font-black text-white">{count}</span>
                    <span className="text-white/60 text-sm">issues</span>
                </div>
            </div>
        </div>
    );
};

interface LandingProps {
    podCount: number;
    deploymentCount: number;
    jobCount: number;
}

export const Landing: React.FC<LandingProps> = ({ podCount, deploymentCount, jobCount }) => {
    const navigate = useNavigate();

    return (
        <div className="min-h-screen bg-gradient-to-br from-gray-900 via-gray-800 to-gray-900 flex flex-col items-center justify-center p-8">
            {/* Title */}
            <div className="text-center mb-16">
                <h1 className="text-6xl font-black text-transparent bg-clip-text bg-gradient-to-r from-purple-400 via-pink-500 to-red-500 mb-4 tracking-tight">
                    AMOS
                </h1>
                <p className="text-gray-400 text-lg">Agentic Mesh Observability System</p>
            </div>

            {/* Category Cards */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-4xl w-full">
                <CategoryCard
                    title="Pods"
                    icon={<Box className="w-10 h-10 text-white" />}
                    count={podCount}
                    color="purple"
                    gradient="bg-gradient-to-br from-purple-600 to-purple-800"
                    onClick={() => navigate('/pods')}
                />
                <CategoryCard
                    title="Deployments"
                    icon={<Layers className="w-10 h-10 text-white" />}
                    count={deploymentCount}
                    color="indigo"
                    gradient="bg-gradient-to-br from-indigo-600 to-indigo-800"
                    onClick={() => navigate('/deployments')}
                />
                <CategoryCard
                    title="Jobs"
                    icon={<BriefcaseBusiness className="w-10 h-10 text-white" />}
                    count={jobCount}
                    color="teal"
                    gradient="bg-gradient-to-br from-teal-600 to-teal-800"
                    onClick={() => navigate('/jobs')}
                />
            </div>

            {/* Footer */}
            <div className="mt-16 text-gray-500 text-sm">
                Monitoring your Kubernetes cluster in real-time
            </div>
        </div>
    );
};
