import { Button, Input } from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';

import { type WorkspaceResponse } from '../../entity/workspaces';

interface Props {
  workspaces: WorkspaceResponse[];
  selectedWorkspace?: WorkspaceResponse;
  onCreateWorkspace: () => void;
  onWorkspaceSelect: (workspace: WorkspaceResponse) => void;
}

export const WorkspaceSelectionComponent = ({
  workspaces,
  selectedWorkspace,
  onCreateWorkspace,
  onWorkspaceSelect,
}: Props) => {
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const dropdownRef = useRef<HTMLDivElement>(null);

  const filteredWorkspaces = useMemo(() => {
    if (!searchValue.trim()) return workspaces;
    const searchTerm = searchValue.toLowerCase();
    return workspaces.filter((workspace) => workspace.name.toLowerCase().includes(searchTerm));
  }, [workspaces, searchValue]);

  const openWorkspace = (workspace: WorkspaceResponse) => {
    setIsDropdownOpen(false);
    setSearchValue('');
    onWorkspaceSelect?.(workspace);
  };

  // Handle click outside dropdown
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsDropdownOpen(false);
        setSearchValue('');
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  if (workspaces.length === 0) {
    return (
      <Button
        type="primary"
        onClick={onCreateWorkspace}
        className="border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700"
      >
        Create workspace
      </Button>
    );
  }

  return (
    <div className="my-1 w-[250px] select-none" ref={dropdownRef}>
      <div className="mb-1 text-xs text-gray-400" style={{ lineHeight: 0.7 }}>
        Selected workspace
      </div>

      <div className="relative">
        {/* Dropdown Trigger */}
        <div
          className="cursor-pointer rounded bg-gray-100 p-1 px-2 hover:bg-gray-200"
          onClick={() => setIsDropdownOpen(!isDropdownOpen)}
        >
          <div className="flex items-center justify-between text-sm">
            <div className="max-w-[250px] truncate">
              {selectedWorkspace?.name || 'Select a workspace'}
            </div>
            <img
              src="/icons/menu/arrow-down-gray.svg"
              alt="arrow-down"
              className={`ml-1 transition-transform duration-200 ${isDropdownOpen ? 'rotate-180' : ''}`}
              width={15}
              height={15}
            />
          </div>
        </div>

        {/* Dropdown Menu */}
        {isDropdownOpen && (
          <div className="absolute top-full left-0 z-50 mt-1 min-w-full rounded-md border border-gray-200 bg-white shadow-lg">
            {/* Search Input */}
            <div className="border-b border-gray-100 p-2">
              <Input
                placeholder="Search workspaces..."
                value={searchValue}
                onChange={(e) => setSearchValue(e.target.value)}
                className="border-0 shadow-none"
                autoFocus
              />
            </div>

            {/* Workspace List */}
            <div className="max-h-[400px] overflow-y-auto">
              {filteredWorkspaces.map((workspace) => (
                <div
                  key={workspace.id}
                  className="max-w-[250px] cursor-pointer truncate px-3 py-2 text-sm hover:bg-gray-50"
                  onClick={() => openWorkspace(workspace)}
                >
                  {workspace.name}
                </div>
              ))}

              {filteredWorkspaces.length === 0 && searchValue && (
                <div className="px-3 py-2 text-sm text-gray-500">No workspaces found</div>
              )}
            </div>

            {/* Create New Workspace Button - Fixed at bottom */}
            <div className="border-t border-gray-100">
              <div
                className="cursor-pointer px-3 py-2 text-sm text-blue-600 hover:bg-gray-50 hover:text-blue-700"
                onClick={() => {
                  onCreateWorkspace();
                  setIsDropdownOpen(false);
                  setSearchValue('');
                }}
              >
                + Create new workspace
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
