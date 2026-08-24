import { useEffect, useState } from "react";

type Task = {
  id: number;
  title: string;
  completed: boolean;
};

function App() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("http://localhost:8080/tasks")
      .then((response) => response.json())
      .then((data) => {
        setTasks(data);
        setLoading(false);
      })
      .catch((error) => {
        console.error("Failed to load tasks:", error);
        setLoading(false);
      });
  }, []);

  if (loading) {
    return <p>Loading...</p>;
  }

  return (
    <div>
      <h1>DevOps Training</h1>

      <h2>Tasks</h2>

      {tasks.map((task) => (
        <div key={task.id}>
          {task.title} - {task.completed ? "Completed" : "Pending"}
        </div>
      ))}
    </div>
  );
}

export default App;