import {
  BarChart, Bar, LineChart, Line, PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend,
} from "recharts";
import type { ChartData } from "@/types/ppt";

interface SlideChartProps {
  data: ChartData;
  colors: string[];
}

const DEFAULT_COLORS = ["#2563EB", "#F97316", "#10B981", "#8B5CF6", "#EF4444", "#06B6D4"];

export default function SlideChart({ data, colors }: SlideChartProps) {
  const palette = colors.length > 0 ? colors : DEFAULT_COLORS;

  if (data.type === "pie" || data.type === "doughnut") {
    const pieData = data.labels.map((label, i) => ({
      name: label,
      value: data.datasets[0]?.values[i] ?? 0,
    }));
    const innerRadius = data.type === "doughnut" ? "40%" : 0;

    return (
      <ResponsiveContainer width="100%" height="100%">
        <PieChart>
          <Pie
            data={pieData}
            dataKey="value"
            nameKey="name"
            cx="50%"
            cy="50%"
            innerRadius={innerRadius}
            outerRadius="70%"
            label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
            labelLine={false}
          >
            {pieData.map((_, i) => (
              <Cell key={i} fill={palette[i % palette.length]} />
            ))}
          </Pie>
          <Tooltip />
        </PieChart>
      </ResponsiveContainer>
    );
  }

  // Bar or Line chart
  const chartData = data.labels.map((label, i) => {
    const point: Record<string, string | number> = { name: label };
    data.datasets.forEach((ds, di) => {
      point[ds.label || `series${di}`] = ds.values[i] ?? 0;
    });
    return point;
  });

  if (data.type === "line") {
    return (
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={chartData} margin={{ top: 10, right: 20, left: 0, bottom: 5 }}>
          <CartesianGrid strokeDasharray="3 3" opacity={0.3} />
          <XAxis dataKey="name" tick={{ fontSize: 10 }} />
          <YAxis tick={{ fontSize: 10 }} />
          <Tooltip />
          {data.datasets.length > 1 && <Legend />}
          {data.datasets.map((ds, i) => (
            <Line
              key={i}
              type="monotone"
              dataKey={ds.label || `series${i}`}
              stroke={palette[i % palette.length]}
              strokeWidth={2}
              dot={{ r: 3 }}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    );
  }

  // Default: bar chart
  return (
    <ResponsiveContainer width="100%" height="100%">
      <BarChart data={chartData} margin={{ top: 10, right: 20, left: 0, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" opacity={0.3} />
        <XAxis dataKey="name" tick={{ fontSize: 10 }} />
        <YAxis tick={{ fontSize: 10 }} />
        <Tooltip />
        {data.datasets.length > 1 && <Legend />}
        {data.datasets.map((ds, i) => (
          <Bar
            key={i}
            dataKey={ds.label || `series${i}`}
            fill={palette[i % palette.length]}
            radius={[2, 2, 0, 0]}
          />
        ))}
      </BarChart>
    </ResponsiveContainer>
  );
}
