const SkeletonChart = () => (
  <div className="bg-chart-bg border border-chart-bar rounded-xl h-52 flex items-end justify-center gap-2 px-4 py-6">
    <div className="w-[8%] bg-gray-200 rounded animate-pulse h-16" />
    <div className="w-[8%] bg-gray-200 rounded animate-pulse h-24" />
    <div className="w-[8%] bg-gray-200 rounded animate-pulse h-20" />
    <div className="w-[8%] bg-gray-200 rounded animate-pulse h-10" />
    <div className="w-[8%] bg-gray-200 rounded animate-pulse h-28" />
    <div className="w-[8%] bg-gray-200 rounded animate-pulse h-14" />
    <div className="w-[8%] bg-gray-200 rounded animate-pulse h-18" />
    <div className="w-[8%] bg-gray-200 rounded animate-pulse h-22" />
  </div>
);

export default SkeletonChart;