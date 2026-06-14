const SkeletonLogsRow = () => (
  <tr className="border-t border-border animate-pulse">
    <td className="px-3.5 py-2.5">
      <div className="h-4 bg-gray-200 rounded w-20" />
    </td>
    <td className="px-3.5 py-2.5">
      <div className="h-4 bg-gray-200 rounded w-48" />
    </td>
    <td className="px-3.5 py-2.5 text-center">
      <div className="h-5 bg-gray-200 rounded w-12 mx-auto" />
    </td>
  </tr>
);

export default SkeletonLogsRow;