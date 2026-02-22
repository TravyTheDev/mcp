import { createFileRoute, Link } from '@tanstack/react-router'
import { useEffect, useState } from 'react'

export const Route = createFileRoute('/')({ component: App })

type Human = {
  id: number;
  firstName: string;
  lastName: string;
  dateOfBirth: string;
  hasAllergies: boolean,
  bio: string;
}

function App() {
  const apiUrl = import.meta.env.VITE_API_URL
  const [humans, setHumans] = useState<Human[]>([])
  
  const loadData = async () => {
    try {
      const res = await fetch(`${apiUrl}/mcp_api/load_humans`)
      const data = await res.json()
      setHumans(data)
    } catch (error) {
      console.log(error)
    }
  }

  useEffect(() => {
    loadData()
  }, [])

  return (
    <div>
      <Link to='/mcp_client'>
        mcp_client
      </Link>
      <table>
        <thead>
          <tr>
            <th>First Name</th>
            <th>Last Name</th>
            <th>DOB</th>
            <th>Allergies?</th>
            <th>Bio</th>
          </tr>
        </thead>
        <tbody>
          {humans.map((human: Human, index: number) => (
            <tr key={index}>
              <td>{human.firstName}</td>
              <td>{human.lastName}</td>
              <td>{human.dateOfBirth}</td>
              <td>{human.hasAllergies ? "⚠️ Yes" : "✅ No"}</td>
              <td>{human.bio}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
