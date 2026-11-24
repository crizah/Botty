import { createContext, useCallback, useContext, useEffect, useState } from 'react';
import api from '../config/axios';

// Projects + groups for the sidebar and dashboard, shared across pages.
// Call refresh() after anything that changes them (create project, attach a group, ...).
const DataContext = createContext(null);

export function DataProvider({ children }) {
  const [projects, setProjects] = useState(null);
  const [groups, setGroups] = useState(null);
  const [newProjectOpen, setNewProjectOpen] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const [p, g] = await Promise.all([api.get('/projects'), api.get('/groups')]);
      setProjects(p.data.projects);
      setGroups(g.data.groups);
    } catch (e) {
      /* a 401 is handled by the axios interceptor; other errors show up on the pages themselves */
    }
  }, []);

  useEffect(() => {
    refresh();
    // TODO: change - polling stands in for the ingest worker pushing newly joined groups
    const t = setInterval(refresh, 30000);
    return () => clearInterval(t);
  }, [refresh]);

  return (
    <DataContext.Provider value={{ projects, groups, refresh, newProjectOpen, setNewProjectOpen }}>
      {children}
    </DataContext.Provider>
  );
}

export const useData = () => useContext(DataContext);
